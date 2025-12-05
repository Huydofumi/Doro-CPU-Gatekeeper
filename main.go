package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
	"unsafe"

	"github.com/getlantern/systray"
	"github.com/shirou/gopsutil/v3/cpu"
)

//go:embed idle_frames/*.ico
var idleFramesFS embed.FS

//go:embed active_frames/*.ico
var activeFramesFS embed.FS

type Flipbook struct {
	frames    [][]byte
	frameRate time.Duration
}

var (
	idleBook   *Flipbook
	activeBook *Flipbook
	currentCPU float64
	logFile    *os.File
	enableLog  bool
)

// Windows API để tạo mutex
var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func createMutex(name string) (uintptr, error) {
	namep, _ := syscall.UTF16PtrFromString(name)
	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(namep)),
	)
	if ret == 0 {
		return 0, err
	}
	// Check if mutex already exists
	if err == syscall.ERROR_ALREADY_EXISTS {
		return ret, err
	}
	return ret, nil
}

func checkSingleInstance() bool {
	mutexName := "Global\\DoroCPUGatekeeperMonitorMutex"
	_, err := createMutex(mutexName)
	if err == syscall.ERROR_ALREADY_EXISTS {
		return false
	}
	return true
}

func initLogger() {
	if !enableLog {
		log.SetOutput(io.Discard)
		return
	}
	
	var err error
	logFile, err = os.OpenFile("doro-spit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("⚠ Không thể tạo log file: %v", err)
		return
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("========================================")
	log.Println("Doro CPU Gatekeeper started")
}

func main() {
	// Parse command line arguments
	logFlag := flag.Bool("l", false, "Enable logging to doro-spit.log")
	flag.Parse()
	
	enableLog = *logFlag
	
	// Check single instance
	if !checkSingleInstance() {
		if enableLog {
			fmt.Println("Doro CPU Gatekeeper is already running!")
		}
		// Show message box on Windows
		showMessageBox("Doro CPU Gatekeeper", "Application is already running!\nCheck your taskbar.")
		os.Exit(1)
	}
	
	// Khởi tạo logger
	initLogger()
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	log.Println("Loading embedded frames...")
	
	// Load pre-extracted ICO frames từ embedded filesystem
	var err error
	idleBook, err = loadFlipbookFromEmbedded(idleFramesFS, "idle_frames", 50*time.Millisecond)
	if err != nil {
		log.Fatalf("❌ Lỗi khi load idle_frames: %v", err)
	}
	log.Printf("✓ Loaded idle_frames: %d frames", len(idleBook.frames))

	activeBook, err = loadFlipbookFromEmbedded(activeFramesFS, "active_frames", 50*time.Millisecond)
	if err != nil {
		log.Fatalf("❌ Lỗi khi load active_frames: %v", err)
	}
	log.Printf("✓ Loaded active_frames: %d frames", len(activeBook.frames))

	log.Println("Starting system tray...")
	systray.Run(onReady, onExit)
}

func onReady() {
	log.Println("System tray initialized")
	
	systray.SetTitle("Doro CPU Gatekeeper")
	systray.SetTooltip("Doro when CPU have 0% usage")

	mCPU := systray.AddMenuItem("CPU: 0%", "Current CPU usage")
	mCPU.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Thoát", "Exit application")

	log.Println("Starting CPU monitoring...")
	go monitorCPU()
	
	log.Println("Starting icon animation...")
	go updateIcon()

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				log.Println("User requested exit")
				systray.Quit()
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			mCPU.SetTitle(fmt.Sprintf("CPU: %.1f%%", currentCPU))
			systray.SetTooltip(fmt.Sprintf("Doro when CPU have %.1f%% usage", currentCPU))
		}
	}()
	
	log.Println("✓ Application ready")
}

func onExit() {
	log.Println("Application exiting...")
}

func showMessageBox(title, message string) {
	// Windows API để hiển thị message box
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)
	
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		0x30, // MB_ICONWARNING | MB_OK
	)
}

func monitorCPU() {
	log.Println("CPU monitor goroutine started")
	for {
		percent, err := cpu.Percent(500*time.Millisecond, false)
		if err != nil {
			log.Printf("⚠ Error getting CPU usage: %v", err)
		} else if len(percent) > 0 {
			currentCPU = percent[0]
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func updateIcon() {
	log.Println("Icon animation goroutine started")
	frameIdx := 0
	var currentBook *Flipbook
	var lastBook *Flipbook
	
	// Minimum delay để tránh update quá nhanh
	const minDelay = 33 * time.Millisecond // ~30 FPS max

	for {
		var speedMultiplier float64

		if currentCPU <= 20 {
			currentBook = idleBook
			speedMultiplier = 1.0
		} else {
			currentBook = activeBook
			speedMultiplier = 0.5 + ((currentCPU-21)/(100-21))*1.5
			if speedMultiplier < 0.5 {
				speedMultiplier = 0.5
			}
			if speedMultiplier > 2.0 {
				speedMultiplier = 2.0
			}
		}

		if currentBook == nil || len(currentBook.frames) == 0 {
			log.Println("⚠ Warning: No frames available")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Reset frame khi đổi book
		if currentBook != lastBook {
			frameIdx = 0
			lastBook = currentBook
			if currentBook == idleBook {
				log.Printf("Switched to idle mode (CPU: %.1f%%)", currentCPU)
			} else {
				log.Printf("Switched to active mode (CPU: %.1f%%, speed: %.2fx)", currentCPU, speedMultiplier)
			}
		}

		if frameIdx >= len(currentBook.frames) {
			frameIdx = 0
		}

		// Set icon với error handling
		if frameIdx < len(currentBook.frames) && len(currentBook.frames[frameIdx]) > 0 {
			systray.SetIcon(currentBook.frames[frameIdx])
		}

		// Tính delay và đảm bảo không quá nhanh
		frameDelay := time.Duration(float64(currentBook.frameRate) / speedMultiplier)
		if frameDelay < minDelay {
			frameDelay = minDelay
		}
		
		frameIdx = (frameIdx + 1) % len(currentBook.frames)

		time.Sleep(frameDelay)
	}
}

func loadFlipbookFromEmbedded(fsys embed.FS, framesDir string, baseFrameRate time.Duration) (*Flipbook, error) {
	// Đọc tất cả entries trong thư mục
	entries, err := fsys.ReadDir(framesDir)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc thư mục embedded '%s': %v", framesDir, err)
	}

	// Lọc và sort các file .ico
	var icoFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ico" {
			// QUAN TRỌNG: embed.FS luôn dùng forward slash, không dùng filepath.Join
			icoFiles = append(icoFiles, framesDir+"/"+entry.Name())
		}
	}

	if len(icoFiles) == 0 {
		return nil, fmt.Errorf("không tìm thấy file ICO nào trong embedded '%s'", framesDir)
	}

	// Sort để đảm bảo thứ tự đúng
	sort.Strings(icoFiles)
	
	log.Printf("Found %d ICO files in embedded %s", len(icoFiles), framesDir)

	book := &Flipbook{
		frames:    make([][]byte, 0, len(icoFiles)),
		frameRate: baseFrameRate,
	}

	// Load từng ICO frame từ embedded filesystem
	for _, icoFile := range icoFiles {
		data, err := fsys.ReadFile(icoFile)
		if err != nil {
			log.Printf("⚠ Cảnh báo: Không thể đọc embedded %s: %v", icoFile, err)
			continue
		}

		// Validate ICO data (basic check)
		if len(data) < 22 {
			log.Printf("⚠ Cảnh báo: %s có vẻ không phải file ICO hợp lệ", icoFile)
			continue
		}

		book.frames = append(book.frames, data)
	}

	if len(book.frames) == 0 {
		return nil, fmt.Errorf("không có frame ICO hợp lệ nào được load từ embedded '%s'", framesDir)
	}

	return book, nil
}