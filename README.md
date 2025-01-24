# kiemtraphatnguoi

Tra cuu phat nguoi tai VN

## Mô tả

KiemTraPhatNguoi là một ứng dụng API đơn giản để tra cứu thông tin vi phạm giao thông cá nhân tại Việt Nam.

Ứng dụng cung cấp thông tin vi phạm giao thông dựa trên biển số xe. Dữ liệu được lấy từ hai nguồn:
  1. Nguồn chính: Một API tra cứu vi phạm (e.g., checkphatnguoi.vn).
  2. Nguồn phụ (fallback): Trang csgt.vn khi không tìm thấy thông tin từ nguồn chính.

Ứng dụng cũng sử dụng công nghệ OCR (nhận diện ký tự từ ảnh) để giải mã captcha, tuy nhiên độ chính xác của OCR có thể thấp trong một số trường hợp.

## Hướng dẫn sử dụng

### Chạy ứng dụng

1. Xây dựng và chạy bằng Docker:

```bash
docker build -t kiemtraphatnguoi .
docker run -p 8080:8080 kiemtraphatnguoi
```

2. Chạy trực tiếp:
  
•	Cài đặt Go (v1.20+).
•	Tải thư viện phụ thuộc:


```bash
go mod tidy && go mod vendor
```
	
•	Chạy ứng dụng:

```bash
go run main.go --logtostderr
```

### Các API có sẵn

1. Tra cứu vi phạm

Endpoint: POST /checkplate

Tham số:
	•	bienso (bắt buộc): Biển số xe cần tra cứu.
	•	loaixe (tùy chọn): Loại phương tiện (giá trị: xemay hoặc oto). Mặc định là oto.

Tra cứu xe máy:

```bash
curl --location --request POST 'localhost:8080/checkplate?bienso=51P139039&loaixe=xemay'
```

Tra cứu ô tô (mặc định):

```bash
curl --location --request POST 'localhost:8080/checkplate?bienso=36A-894.42'
```

Kết quả trả về (JSON):

```json
[
  {
    "Biển kiểm soát": "36A-894.42",
    "Hành vi vi phạm": "12321.5.5.i.01.Điều khiển xe chạy quá tốc độ quy định từ 10 km/h đến 20 km/h",
    "Loại phương tiện": "Ô tô",
    "Màu biển": "Nền mầu trắng, chữ và số màu đen",
    "Nơi giải quyết vụ việc": [
      "1. Tỉnh Thanh Hóa",
      "Địa chỉ: Phía bắc, đường tránh Quốc Lộ 1A, Phường Đông Thọ, TP Thanh Hóa",
      "Số điện thoại liên hệ: 02373.853085",
      "2. Đội Cảnh sát giao thông, Trật tự - Công an thành phố Thanh Hóa - Tỉnh Thanh Hóa",
      "Địa chỉ: TP Thanh Hóa"
    ],
    "Thời gian vi phạm": "10:32, 20/12/2024",
    "Trạng thái": "Chưa xử phạt",
    "Đơn vị phát hiện vi phạm": "Tỉnh Thanh Hóa",
    "Địa điểm vi phạm": "Km 64+600, Quốc lộ 45 Địa bàn Tỉnh Thanh Hóa"
  }
]
```

### Lưu ý về giải captcha

Ứng dụng sử dụng một thư viện OCR (e.g., gosseract) để giải mã captcha từ csgt.vn. Tuy nhiên:
•	Độ chính xác thấp: Kết quả giải mã captcha có thể không chính xác với hình ảnh phức tạp.
•	Cách khắc phục: Cần thực hiện nhiều lần request để nhận được captcha dễ giải mã hơn.

	Ghi chú: Ảnh captcha được lưu trong thư mục captchaImageLogs để phục vụ kiểm tra và cải thiện hiệu quả OCR.


