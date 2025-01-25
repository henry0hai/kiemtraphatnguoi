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
curl --location --request POST 'localhost:8080/checkplate?bienso=98E1-714.78&loaixe=xemay'
```

Kết quả trả về (JSON): parse từ HTML

```json
[
  {
    "plate": "98E1-714.78",
    "plate_color": "Nền mầu trắng, chữ và số màu đen",
    "vehicle_type": "Xe máy",
    "violation_time": "14:52, 06/01/2025",
    "violation_place": "Ngã 4 Trần Nguyên Hãn - Trần Quang Khải, Phường Thọ Xương, Thành phố Bắc Giang, Tỉnh Bắc Giang",
    "violation_action": "16824.7.2.h.01.Không đội “mũ bảo hiểm cho người đi mô tô, xe máy” khi điều khiển xe tham gia giao thông trên đường bộ",
    "status": "Chưa xử phạt",
    "detected_by": "Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang",
    "resolution_location": "1. Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang\nĐịa chỉ: số 384 đường Xương Giang, phường Ngô Quyền\nSố điện thoại liên hệ: 0911595121\n2. Đội Cảnh sát giao thông, Trật tự - Công an huyện Lục Ngạn - Tỉnh Bắc Giang\nĐịa chỉ: huyện Lục Ngạn"
  }
]
```


Tra cứu ô tô (mặc định):

```bash
curl --location --request POST 'localhost:8080/checkplate?bienso=98A-290.11'
```

Kết quả trả về (JSON):

```json
[
  {
    "plate": "98A-290.11",
    "plate_color": "Nền mầu trắng, chữ và số màu đen",
    "vehicle_type": "Ô tô",
    "violation_time": "15:03, 06/01/2025",
    "violation_place": "Đường Nguyễn Thị Minh Khai, Phường Xương Giang, Thành phố Bắc Giang, Tỉnh Bắc Giang",
    "violation_action": "16824.6.1.a.04.Không chấp hành hiệu lệnh, chỉ dẫn của vạch kẻ đường",
    "status": "Chưa xử phạt",
    "detected_by": "Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang",
    "resolution_location": "1. Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang\nĐịa chỉ: số 384 đường Xương Giang, phường Ngô Quyền\nSố điện thoại liên hệ: 0911595121"
  },
  {
    "plate": "98A-290.11",
    "plate_color": "Nền mầu trắng, chữ và số màu đen",
    "vehicle_type": "Ô tô",
    "violation_time": "11:38, 15/08/2024",
    "violation_place": "Ngã 4 Xương Giang - Vương Văn Trà - Quang Trung, Phường Trần Phú, Thành phố Bắc Giang, Tỉnh Bắc Giang",
    "violation_action": "12321.5.3.k.06.Điều khiển xe rẽ trái tại nơi có biển báo hiệu có nội dung cấm rẽ trái đối với loại phương tiện đang điều khiển",
    "status": "Chưa xử phạt",
    "detected_by": "Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang",
    "resolution_location": "1. Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang\nĐịa chỉ: số 384 đường Xương Giang, phường Ngô Quyền\nSố điện thoại liên hệ: 0911595121"
  },
  {
    "plate": "98A-290.11",
    "plate_color": "Nền mầu trắng, chữ và số màu đen",
    "vehicle_type": "Ô tô",
    "violation_time": "14:44, 16/10/2023",
    "violation_place": "Ngã 4 Trần Nguyên Hãn - Trần Quang Khải, Phường Thọ Xương, Thành phố Bắc Giang, Tỉnh Bắc Giang",
    "violation_action": "12321.5.5.a.01.Không chấp hành hiệu lệnh của đèn tín hiệu giao thông",
    "status": "Chưa xử phạt",
    "detected_by": "Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang",
    "resolution_location": "1. Đội Cảnh sát giao thông, Trật tự - Công an thành phố Bắc Giang - Tỉnh Bắc Giang\nĐịa chỉ: số 384 đường Xương Giang, phường Ngô Quyền\nSố điện thoại liên hệ: 0911595121"
  }
]
```

### Lưu ý về giải captcha

Ứng dụng sử dụng một thư viện OCR (e.g., gosseract) để giải mã captcha từ csgt.vn. Tuy nhiên:

- Độ chính xác thấp: Kết quả giải mã captcha có thể không chính xác với hình ảnh phức tạp.
- Cách khắc phục: Cần thực hiện nhiều lần request để nhận được captcha dễ giải mã hơn.

> Ghi chú: Ảnh captcha được lưu trong thư mục captchaImageLogs để phục vụ kiểm tra và cải thiện hiệu quả OCR.

## Frontend

[kiemtraphatnguoi-ui](https://github.com/henry0hai/kiemtraphatnguoi-ui)

## Data to test

[data-test](https://nguoiquansat.vn/127-chu-xe-co-bien-so-sau-day-nhanh-chong-den-nop-phat-nguoi-theo-quy-dinh-193380.html)

## Sample Demo (will be unvalidated after a day)

[demo](https://a12e-2001-ee0-d789-ac50-1cf4-4e8c-8a48-aaf8.ngrok-free.app/)

> Remember to retrieve if the response is invalid, until some empty response is appearied.
