# Doro-CPU-Gatekeeper

![https://media.giphy.com/media/vFKqnCdLPNOKc/giphy.gif](https://github.com/Huydofumi/Doro-CPU-Gatekeeper/blob/main/showcase.gif)

Sản phẩm được vai cốt từ claude 4.5 sonnet

Repo này không bao gồm file animation gốc, nhưng file .exe để [download](https://github.com/Huydofumi/Doro-CPU-Gatekeeper/releases/tag/release) về thì đã gồm sẵn file chỉ cần chạy file .exe và enjoy.

# Cơ chế hoạt động

* Khi CPU dưới hoặc bằng 20% sẽ play sequence của file animation 1 đã được extract toàn bộ frame.

* Khi CPU trên 20% dưới 90% sẽ play sequence của file animation 2 đã được extract toàn bộ frame và giao động từ 50% speed đến 200% speed của sequence đó, để có thể biết rằng CPU đang bị heavy load hoặc là đang thảnh thơi dã ngoại.

* Khi CPU trên 90% sẽ play sequence của file animation 3 đã được extract toàn bộ frame.

# Tự sửa animation

* Bạn cần 2 file animation ( A và B và C ), A là sử dụng khi CPU dưới 20%, B là trên 20%, C là trên 90% và nên bỏ chung cùng folder project.

* Định dạng của cả 2 file đều phải là MP4, được resize trước ở 32x32 pixel, 30fps (Có thể dùng after effect để resize cho tiện)

* Cài đặt [golang](https://go.dev/dl/)

* Vào folder project mở cmd lên và gõ các dòng sau để download library về

  `go mod init yourprojectname`
  
  `go get github.com/getlantern/systray`
  
  `go get github.com/shirou/gopsutil/v3/cpu`
  
  `go mod tidy`

* Sử dụng frame_extract.go với file animation A với câu lệnh trong CMD ( Đã trỏ vào thư mục )
  
  `go run extract_frames.go A.mp4 idle_frames`
  
  > _`idle_frames` là bắt buộc hoặc bạn có thể tự sửa code cho nó thành cái gì đó khác_
  
* Sử dụng frame_extract.go với file animation B với câu lệnh trong CMD ( Đã trỏ vào thư mục )

  `go run extract_frames.go B.mp4 active_frames`
  
  > _`active_frames` là bắt buộc hoặc bạn có thể tự sửa code cho nó thành cái gì đó khác_

* Sử dụng frame_extract.go với file animation C với câu lệnh trong CMD ( Đã trỏ vào thư mục )

  `go run extract_frames.go C.mp4 heavy_active_frames`
  
  > _`heavy_active_frames` là bắt buộc hoặc bạn có thể tự sửa code cho nó thành cái gì đó khác_

* Sau khi đã có 2 thư mục với frame đã được extract thì build binary với câu lệnh sau

  `go build -ldflags -H=windowsgui -o yourapplicationname.exe main.go`
  > _Thay `yourapplicationname` bằng tên ứng dụng bạn tuỳ thích_

* Chạy file .exe và thưởng thức
