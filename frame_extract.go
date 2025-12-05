package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run extract_frames.go <video_file> [output_dir] [size]")
		fmt.Println("Ví dụ: go run extract_frames.go idle.mp4 idle_frames 16")
		fmt.Println("       go run extract_frames.go active.mp4 active_frames 16")
		return
	}

	videoFile := os.Args[1]
	outputDir := "frames"
	size := 32 // default size for system tray
	
	if len(os.Args) >= 3 {
		outputDir = os.Args[2]
	}
	
	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[3], "%d", &size)
	}

	fmt.Printf("Converting %s to ICO frames (%dx%d)...\n", videoFile, size, size)
	
	// Check ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.Fatal("❌ ffmpeg không được cài đặt. Cài đặt: https://ffmpeg.org/download.html")
	}

	// Tạo temp directory
	tempDir := "temp_extract"
	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Tạo output directory
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)

	// Extract frames từ video bằng ffmpeg
	fmt.Println("📹 Đang extract frames từ video...")
	cmd := exec.Command("ffmpeg",
		"-i", videoFile,
		"-vf", fmt.Sprintf("scale=%d:%d:flags=lanczos", size, size),
		"-vsync", "0",
		filepath.Join(tempDir, "frame_%04d.png"),
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("❌ Lỗi ffmpeg: %v\nOutput: %s", err, string(output))
	}

	// Đọc các PNG frames
	files, err := filepath.Glob(filepath.Join(tempDir, "frame_*.png"))
	if err != nil || len(files) == 0 {
		log.Fatal("❌ Không tìm thấy frames được extract")
	}

	sort.Strings(files)
	fmt.Printf("✓ Đã extract %d frames\n", len(files))
	fmt.Println("🔄 Đang convert sang ICO...")

	// Convert từng frame sang ICO
	successCount := 0
	for i, pngFile := range files {
		// Load PNG
		f, err := os.Open(pngFile)
		if err != nil {
			log.Printf("⚠ Lỗi mở %s: %v", pngFile, err)
			continue
		}

		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			log.Printf("⚠ Lỗi decode PNG %s: %v", pngFile, err)
			continue
		}

		// Convert sang RGBA nếu cần
		var rgba *image.RGBA
		if rgbaImg, ok := img.(*image.RGBA); ok {
			rgba = rgbaImg
		} else {
			bounds := img.Bounds()
			rgba = image.NewRGBA(bounds)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					rgba.Set(x, y, img.At(x, y))
				}
			}
		}

		// Generate ICO
		iconData, err := generateICO(rgba)
		if err != nil {
			log.Printf("⚠ Lỗi tạo ICO frame %d: %v", i, err)
			continue
		}

		// Save ICO với tên frame_0000.ico, frame_0001.ico, ...
		outputFile := filepath.Join(outputDir, fmt.Sprintf("frame_%04d.ico", i))
		err = os.WriteFile(outputFile, iconData, 0644)
		if err != nil {
			log.Printf("⚠ Lỗi lưu %s: %v", outputFile, err)
			continue
		}

		successCount++
		if (i+1)%10 == 0 || i == len(files)-1 {
			fmt.Printf("  Progress: %d/%d frames\r", i+1, len(files))
		}
	}

	fmt.Printf("\n✅ Hoàn tất! %d frames được lưu vào thư mục: %s/\n", successCount, outputDir)
	fmt.Printf("📦 Khi build app, hãy đặt thư mục '%s' cùng thư mục với file .exe\n", outputDir)
}

func generateICO(img *image.RGBA) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	if width > 256 || height > 256 {
		return nil, fmt.Errorf("image quá lớn: %dx%d (max 256x256)", width, height)
	}
	
	var buf bytes.Buffer

	// ICO Header (6 bytes)
	binary.Write(&buf, binary.LittleEndian, uint16(0))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(1))

	// Icon Directory Entry (16 bytes)
	widthByte := byte(width)
	heightByte := byte(height)
	if width == 256 {
		widthByte = 0
	}
	if height == 256 {
		heightByte = 0
	}
	
	buf.WriteByte(widthByte)
	buf.WriteByte(heightByte)
	buf.WriteByte(0)
	buf.WriteByte(0)
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(32))
	
	maskRowSize := (width + 31) / 32 * 4
	bitmapSize := uint32(40 + width*height*4 + height*maskRowSize)
	binary.Write(&buf, binary.LittleEndian, bitmapSize)
	binary.Write(&buf, binary.LittleEndian, uint32(22))

	// BITMAPINFOHEADER (40 bytes)
	binary.Write(&buf, binary.LittleEndian, uint32(40))
	binary.Write(&buf, binary.LittleEndian, int32(width))
	binary.Write(&buf, binary.LittleEndian, int32(height*2))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(32))
	binary.Write(&buf, binary.LittleEndian, uint32(0))
	binary.Write(&buf, binary.LittleEndian, uint32(width*height*4))
	binary.Write(&buf, binary.LittleEndian, int32(0))
	binary.Write(&buf, binary.LittleEndian, int32(0))
	binary.Write(&buf, binary.LittleEndian, uint32(0))
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// Write pixel data (bottom-up, BGRA format)
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			alpha := byte(a >> 8)
			if alpha == 0 {
				buf.WriteByte(0)
				buf.WriteByte(0)
				buf.WriteByte(0)
				buf.WriteByte(0)
			} else {
				buf.WriteByte(byte(b >> 8))
				buf.WriteByte(byte(g >> 8))
				buf.WriteByte(byte(r >> 8))
				buf.WriteByte(alpha)
			}
		}
	}

	// AND mask
	for y := 0; y < height; y++ {
		for x := 0; x < maskRowSize; x++ {
			buf.WriteByte(0x00)
		}
	}

	return buf.Bytes(), nil
}