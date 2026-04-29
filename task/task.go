package task

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/buzhiyun/finance-invoice/zhipu"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

const (
	MaxRetries     = 10
	baseRetryDelay = 2 * time.Second
	maxRetryDelay  = 5 * time.Minute
)

type InvoiceFields = zhipu.InvoiceFields

type PageResult struct {
	PageNum    int           `json:"page_num"`
	Status     string        `json:"status"`
	RetryCount int           `json:"retry_count"`
	Fields     InvoiceFields `json:"fields"`
	Error      string        `json:"error,omitempty"`
	PageFile   string        // temp file path, not serialized
	PDFText    []string      // PDF raw text for validation, not serialized
}

type FileTask struct {
	Filename string        `json:"filename"`
	Pages    []*PageResult `json:"pages"`
}

type BatchTask struct {
	ID        string      `json:"id"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	Files     []*FileTask `json:"files"`
	ExcelPath string      `json:"excel_path,omitempty"`
	Error     string      `json:"error,omitempty"`
}

func (t *BatchTask) MarshalJSON() ([]byte, error) {
	type Alias BatchTask
	return json.Marshal(&struct {
		HasExcel bool `json:"has_excel"`
		*Alias
	}{
		HasExcel: t.ExcelPath != "",
		Alias:    (*Alias)(t),
	})
}

// ExcelGenerator abstracts Excel generation to avoid circular imports.
type ExcelGenerator interface {
	Generate(t *BatchTask, outputPath string) error
}

type Manager struct {
	mu        sync.RWMutex
	tasks     map[string]*BatchTask
	zhipu     *zhipu.Client
	excelGen  ExcelGenerator
	semaphore chan struct{}
	outputDir string
	tmpDir    string
}

func NewManager(zhipuClient *zhipu.Client, excelGen ExcelGenerator, maxConcurrent int) (*Manager, error) {
	outputDir := "output"
	tmpDir := "tmp"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("create tmp dir: %w", err)
	}

	return &Manager{
		tasks:     make(map[string]*BatchTask),
		zhipu:     zhipuClient,
		excelGen:  excelGen,
		semaphore: make(chan struct{}, maxConcurrent),
		outputDir: outputDir,
		tmpDir:    tmpDir,
	}, nil
}

func (tm *Manager) CreateTask(files map[string][]byte) *BatchTask {
	id := fmt.Sprintf("%d", time.Now().UnixMilli())

	batchTask := &BatchTask{
		ID:        id,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	for filename, data := range files {
		fileTask := &FileTask{
			Filename: filename,
		}

		pagePaths, err := splitPDFPages(data, tm.tmpDir, id, filename)
		if err != nil {
			fileTask.Pages = append(fileTask.Pages, &PageResult{
				PageNum: 1,
				Status:  "failed",
				Error:   fmt.Sprintf("PDF split failed: %v", err),
			})
			batchTask.Files = append(batchTask.Files, fileTask)
			continue
		}

		for i, pp := range pagePaths {
			fileTask.Pages = append(fileTask.Pages, &PageResult{
				PageNum:  i + 1,
				Status:   "pending",
				Fields:   InvoiceFields{},
				PageFile: pp,
			})
		}
		batchTask.Files = append(batchTask.Files, fileTask)
	}

	tm.mu.Lock()
	tm.tasks[id] = batchTask
	tm.mu.Unlock()

	totalPages := 0
	for _, ft := range batchTask.Files {
		totalPages += len(ft.Pages)
	}
	log.Printf("[Task %s] 创建任务, %d 个文件, %d 页待处理", id, len(batchTask.Files), totalPages)

	go tm.processTask(batchTask)

	return batchTask
}

func (tm *Manager) processTask(task *BatchTask) {
	tm.updateTaskStatus(task, "processing")
	log.Printf("[Task %s] 开始处理", task.ID)

	var wg sync.WaitGroup
	for _, ft := range task.Files {
		for _, pr := range ft.Pages {
			if pr.Status == "failed" {
				log.Printf("[Task %s] %s 第%d页 跳过(PDF拆分失败): %s", task.ID, ft.Filename, pr.PageNum, pr.Error)
				continue
			}
			wg.Add(1)
			go func(ft *FileTask, pr *PageResult) {
				defer wg.Done()
				tm.processPage(task.ID, ft.Filename, pr)
			}(ft, pr)
		}
	}
	wg.Wait()

	hasFailed := false
	allFailed := true
	for _, ft := range task.Files {
		for _, pr := range ft.Pages {
			switch pr.Status {
			case "success":
				allFailed = false
			case "failed":
				hasFailed = true
			}
		}
	}

	if allFailed {
		tm.updateTaskStatus(task, "failed")
		log.Printf("[Task %s] 全部失败", task.ID)
		return
	}

	excelPath := filepath.Join(tm.outputDir, fmt.Sprintf("invoice_%s.xlsx", task.ID))
	if err := tm.excelGen.Generate(task, excelPath); err != nil {
		task.Error = fmt.Sprintf("generate excel failed: %v", err)
		tm.updateTaskStatus(task, "failed")
		log.Printf("[Task %s] Excel生成失败: %v", task.ID, err)
		return
	}

	task.ExcelPath = excelPath
	if hasFailed {
		tm.updateTaskStatus(task, "partial_failed")
		log.Printf("[Task %s] 部分完成(有失败页), Excel已生成: %s", task.ID, excelPath)
	} else {
		tm.updateTaskStatus(task, "completed")
		log.Printf("[Task %s] 全部完成, Excel已生成: %s", task.ID, excelPath)
	}
}

func (tm *Manager) processPage(taskID, filename string, pr *PageResult) {
	tm.semaphore <- struct{}{}
	defer func() { <-tm.semaphore }()

	pr.Status = "processing"
	pageTag := fmt.Sprintf("[Task %s] %s 第%d页", taskID, filename, pr.PageNum)
	log.Printf("%s 开始处理", pageTag)

	if pr.PageFile == "" {
		pr.Status = "failed"
		pr.Error = "temp file not found"
		log.Printf("%s 失败: 临时文件不存在", pageTag)
		return
	}

	pageData, err := os.ReadFile(pr.PageFile)
	if err != nil {
		pr.Status = "failed"
		pr.Error = fmt.Sprintf("read page file: %v", err)
		log.Printf("%s 失败: 读取临时文件出错: %v", pageTag, err)
		return
	}

	pr.PDFText = extractPDFText(pr.PageFile)

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			delay := min(baseRetryDelay*time.Duration(1<<uint(attempt-1)), maxRetryDelay)
			log.Printf("%s 第%d次重试, 等待 %v", pageTag, attempt, delay)
			time.Sleep(delay)
		}

		url, err := tm.zhipu.UploadFile(filepath.Base(pr.PageFile), pageData)
		if err != nil {
			lastErr = fmt.Errorf("upload failed: %w", err)
			pr.RetryCount = attempt
			log.Printf("%s 上传失败(第%d次): %v", pageTag, attempt+1, err)
			continue
		}
		log.Printf("%s 上传成功, URL: %s", pageTag, url)

		fields, err := tm.zhipu.RecognizeInvoice(url)
		if err != nil {
			lastErr = fmt.Errorf("recognize failed: %w", err)
			pr.RetryCount = attempt
			log.Printf("%s 识别失败(第%d次): %v", pageTag, attempt+1, err)
			continue
		}

		pr.Status = "success"
		pr.RetryCount = attempt
		pr.Fields = *fields
		log.Printf("%s 识别成功: 类型=%s, 号码=%s, 日期=%s, 购方=%s, 销方=%s, 金额=%s, 税率=%s, 税额=%s, 价税合计=%s",
			pageTag, fields.InvoiceType, fields.InvoiceNumber, fields.InvoiceDate,
			fields.BuyerName, fields.SellerName, fields.Amount,
			fields.TaxRate, fields.TaxAmount, fields.TotalLower)
		return
	}

	pr.Status = "failed"
	pr.RetryCount = MaxRetries
	pr.Error = lastErr.Error()
	log.Printf("%s 最终失败(重试%d次): %v", pageTag, MaxRetries, lastErr)
}

func (tm *Manager) updateTaskStatus(task *BatchTask, status string) {
	tm.mu.Lock()
	task.Status = status
	tm.mu.Unlock()
}

func (tm *Manager) GetTask(id string) *BatchTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tasks[id]
}

func (tm *Manager) ListTasks() []*BatchTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*BatchTask, 0, len(tm.tasks))
	for _, t := range tm.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (tm *Manager) ClearFinished() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	removed := 0
	for id, t := range tm.tasks {
		if t.Status == "completed" || t.Status == "failed" || t.Status == "partial_failed" {
			delete(tm.tasks, id)
			removed++
		}
	}
	log.Printf("[TaskManager] 清理了 %d 个已结束任务", removed)
	return removed
}

func splitPDFPages(data []byte, tmpDir, taskID, filename string) ([]string, error) {
	srcPath := filepath.Join(tmpDir, fmt.Sprintf("%s_%s", taskID, filename))
	if err := os.WriteFile(srcPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write temp pdf: %w", err)
	}
	defer os.Remove(srcPath)

	safeName := filename
	if len(safeName) > 50 {
		safeName = safeName[:50]
	}
	pageDir := filepath.Join(tmpDir, fmt.Sprintf("%s_%s_pages", taskID, safeName))
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		return nil, fmt.Errorf("create page dir: %w", err)
	}

	if err := api.ExtractPagesFile(srcPath, pageDir, nil, nil); err != nil {
		return nil, fmt.Errorf("extract pages: %w", err)
	}

	entries, err := os.ReadDir(pageDir)
	if err != nil {
		return nil, fmt.Errorf("read page dir: %w", err)
	}

	var pageFiles []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".pdf" {
			continue
		}
		pageFiles = append(pageFiles, filepath.Join(pageDir, e.Name()))
	}

	if len(pageFiles) == 0 {
		return nil, fmt.Errorf("no pages extracted from %s", filename)
	}

	slices.Sort(pageFiles)

	return pageFiles, nil
}
