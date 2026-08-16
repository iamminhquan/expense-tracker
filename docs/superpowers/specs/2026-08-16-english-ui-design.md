# Chuyển giao diện sang tiếng Anh

Ngày: 2026-08-16
Trạng thái: đã duyệt design, chờ review spec

## Mục tiêu

Toàn bộ chữ người dùng nhìn thấy chuyển sang tiếng Anh: template, thông báo
lỗi từ handler, nhãn tháng, tên 9 danh mục mặc định, và quy ước định dạng
tiền/ngày.

Chức năng chuyển đổi ngôn ngữ (language switcher) **không** nằm trong phạm vi
lần này. Nhưng một quyết định ở đây phải tính trước cho nó: tên danh mục mặc
định nằm trong DB chứ không nằm trong template, nên nếu lưu thẳng tiếng Anh
vào cột `name` thì lúc làm switcher sẽ phải làm lại đúng công việc này. Vì vậy
spec thêm cột `slug` ngay bây giờ — đây là phần duy nhất được làm "sẵn cho
tương lai", mọi thứ khác hardcode tiếng Anh theo tinh thần YAGNI.

## Ngoài phạm vi

- Language switcher, cột `users.language`, catalog tiếng Việt.
- Đưa ~200 chuỗi UI vào catalog i18n. Template viết thẳng tiếng Anh.
- Dịch `docs/design/design_handoff_expense_tracker/SPEC.md` — đó là tài liệu
  handoff thiết kế gốc, dịch đi là mất dấu vết. Chỉ thêm ghi chú quy ước mới.
- Tên danh mục do người dùng tự tạo. Đó là dữ liệu của họ, giữ nguyên.

## 1. Schema: migration `000008_add_category_slug`

Bảng `categories` hiện có `(id, user_id, name, type, color, created_at)`.
Hàng mặc định (dùng chung mọi tài khoản) là hàng có `user_id IS NULL`.

### Up

```sql
ALTER TABLE categories ADD COLUMN slug TEXT;
```

Rồi `UPDATE` từng hàng mặc định, khớp theo `name` tiếng Việt hiện tại, đặt
đồng thời `slug` và `name` mới:

| `name` cũ | `slug` | `name` mới | type |
|---|---|---|---|
| Ăn uống | `food_drink` | Food & Drink | expense |
| Đi lại | `transport` | Transport | expense |
| Giải trí | `entertainment` | Entertainment | expense |
| Hóa đơn | `bills` | Bills | expense |
| Sức khỏe | `health` | Health | expense |
| Mua sắm | `shopping` | Shopping | expense |
| Khác | `other` | Other | expense |
| Lương | `salary` | Salary | income |
| Thưởng | `bonus` | Bonus | income |
| Thu nhập khác | `other_income` | Other income | income |

```sql
CREATE UNIQUE INDEX idx_categories_slug ON categories (slug) WHERE slug IS NOT NULL;
```

### Vì sao đặt cả `slug` lẫn `name` tiếng Anh

`slug` một mình là đủ để hiển thị, nhưng không đủ cho phần còn lại của hệ
thống:

- `ListCategoriesForUser` và `ListCategoriesWithTransactionCounts` sắp xếp
  `ORDER BY c.user_id NULLS FIRST, c.name`. Nếu `name` còn tiếng Việt mà UI
  hiển thị bản dịch từ slug, thứ tự trên màn hình sẽ theo Ă-Đ-G-H-L-M-S-T,
  không theo thứ tự chữ cái tiếng Anh người dùng đang nhìn.
- `idx_categories_user_type_name` (unique trên `user_id, type, name`) chặn
  một người dùng tạo hai danh mục trùng tên. Nếu `name` mặc định còn tiếng
  Việt, không có gì chặn họ tạo danh mục riêng tên "Transport" trùng với
  danh mục mặc định đang hiển thị là Transport.

Nên `name` giữ vai trò nhãn ngôn ngữ mặc định, `slug` giữ vai trò khóa ngữ
nghĩa. Hôm nay hai thứ trùng nội dung; khi có catalog tiếng Việt thì slug
thắng ở tầng hiển thị còn `name` vẫn phục vụ sort/unique.

### `other_income` — hàng dễ bị bỏ sót

Migration `000006` xóa `Thu nhập khác` **chỉ khi** chưa có giao dịch nào trỏ
vào nó (xóa khi đang được tham chiếu sẽ vi phạm FK của `transactions`). Nên
database mới không còn hàng này, nhưng tài khoản đã dùng nó thì vẫn còn. Phải
gán slug cho nó, nếu không nó là hàng mặc định duy nhất rơi lại tiếng Việt.
`UPDATE` không khớp hàng nào là no-op, an toàn cho cả hai trường hợp.

### Down

Drop index, `UPDATE` ngược `name` về tiếng Việt khớp theo `slug`, rồi
`ALTER TABLE categories DROP COLUMN slug`. Down migration này đảo ngược được
trọn vẹn, khác `000006` (vốn best-effort vì có bước lossy).

## 2. Dịch tên danh mục khi render

File mới `internal/i18n/categories.go`:

```go
package i18n

// CategoryNames ánh xạ slug của danh mục mặc định sang nhãn tiếng Anh.
var categoryNames = map[string]string{"food_drink": "Food & Drink", ...}

// CategoryName trả nhãn hiển thị: bản dịch nếu là danh mục mặc định có slug
// nhận diện được, ngược lại trả name do người dùng tự đặt.
func CategoryName(slug pgtype.Text, name string) string
```

Slug rỗng (danh mục người dùng tạo) hoặc slug lạ (hàng mặc định tương lai
chưa có trong map) đều rơi về `name` — không bao giờ trả chuỗi rỗng.

Thêm `NameForSlug(slug string) string` cho chỗ chỉ có slug trong tay, và
`CategoryName` gọi vào nó.

Tên danh mục đến với template qua **ba đường khác nhau**, không phải một —
đây là chỗ dễ làm sót nhất của cả spec này:

**(a) Hàng `sqlcgen.Category`** — có sẵn `Slug` sau migration. Dùng template
func `catName` (đăng ký trong `handlers.TemplateFuncs()`), thay `{{.Name}}` →
`{{catName .Slug .Name}}`:

- `category_row.html:5` (tên trong danh sách), `:33` (hộp thoại xác nhận xóa)
- `transactions.html:154`, `transaction_row.html:65` (`<option>` trong select)
- `transaction_row.html:93` (chip chọn danh mục ở mobile)

**(b) Hàng join mang `category_name`** — `internal/database/queries/transactions.sql`
có ba query `SELECT ... c.name AS category_name` (dòng 2, 33, 55). Các query
này phải thêm `c.slug AS category_slug`, riêng `CategoryBreakdown` (dòng 33)
thêm cả vào `GROUP BY c.slug, c.name, c.color`. Sau đó:

- `transaction_row.html:7,11` → `{{catName .CategorySlug .CategoryName}}`
- Chú giải biểu đồ tròn `dashboard.html:62` **không** dùng được `catName`:
  nó render `pieLegendEntry`, một struct tổng hợp trong Go, không phải hàng
  DB. Bản dịch phải làm ở `buildPieData` khi dựng entry — vốn cũng bắt buộc,
  vì cùng hàm đó dựng `pieLabelsJSON` cho Chart.js và nhãn trong JSON thì
  template func không với tới được.

**(c) `categories.html:68`** — `{{.CategoryName}}` ở đây **không phải** tên
danh mục lấy từ DB mà là giá trị người dùng vừa gõ vào form thêm danh mục,
giữ lại khi form render lỗi. Trùng tên field với (b) hoàn toàn tình cờ. Giữ
nguyên.

Giữ nguyên `{{.Name}}` ở `auth_card_body.html:15` — chỗ đó là tên người dùng.

`category_row.html:70` là ô input sửa tên: chỉ hiện với danh mục người dùng
tự tạo (mặc định không sửa được tên), nên vẫn dùng `{{.Name}}` thô. Nếu dùng
`catName` ở đây, người dùng sẽ sửa một chuỗi đã dịch rồi lưu đè vào `name`.

## 3. Query phụ thuộc tên tiếng Việt

`GetDefaultCategoryForReassignment` trong `internal/database/queries/categories.sql`
đang tìm danh mục nhận giao dịch mồ côi bằng `name = 'Khác'`. Đổi thành
`slug = 'other'` (bỏ luôn điều kiện `type = 'expense'` vì slug đã unique).

Đây là lý do cột slug đáng giá ngay cả khi chưa có switcher: hiện tại đổi
nhãn hiển thị của một danh mục mặc định sẽ làm hỏng luồng xóa danh mục.

## 4. Định dạng — `internal/handlers/format.go`

| Hàm | Trước | Sau |
|---|---|---|
| `formatThousands` | `50.000` | `50,000` |
| `vnd` | `50.000₫` | `50,000₫` |
| `dateFull` | `11/08/2026` | `11 Aug 2026` |
| `dateShort` | `11/08` | `11 Aug` |
| `monthLabel` | `Tháng 8, 2026` | `August 2026` |
| `monthLabelLower` | `tháng 8` | `August` |

Tháng viết chữ thay vì `dd/mm` để tránh nhập nhằng với `mm/dd` — cùng một
chuỗi `11/08` đọc theo hai kiểu ra hai ngày khác nhau và không có cách nào
phân biệt từ chính chuỗi đó.

`vnd` giữ `₫` liền sau số, không đổi sang `VND` hay đưa ký hiệu ra trước.
Dấu của `vndSigned`/`vndBalance` không đổi.

`comparisonText` (`report_handlers.go`): `Tháng trước 12.450.000₫ · giảm 8%`
→ `Last month 12,450,000₫ · down 8%`; các nhánh còn lại thành
`No data for last month` / `Last month 12,450,000₫` / `Last month 12,450,000₫ · unchanged`.

`comparisonTextMobile`: `Giảm 8% so với tháng trước` → `Down 8% vs last month`;
`Không đổi so với tháng trước` → `Unchanged vs last month`.

Cột `Date` trong bảng giao dịch rộng thêm vài px vì `11 Aug` dài hơn `11/08`;
điều chỉnh khi implement nếu layout bị chật.

## 5. Template và chuỗi trong handler

Dịch toàn bộ chuỗi hiển thị trong 7 template và ~30 chuỗi lỗi trong
`auth_handlers.go`, `category_handlers.go`, `transaction_handlers.go`.

`<html lang="vi">` → `lang="en"`.

`<title>` hiện là `Sổ chi tiêu` → `$pend — Expenses`, khép lại điểm còn treo
từ lần thay logo (lúc đó giữ nguyên vì tiêu đề đang đóng vai tên app tiếng
Việt; giờ tên app tiếng Việt không còn chỗ đứng).

Nhãn `Khác` tổng hợp của biểu đồ tròn (`report_handlers.go:141,145` — phần
gộp mọi danh mục ngoài top 5) dùng `i18n.NameForSlug("other")`, không hardcode
chuỗi `"Other"` lần thứ hai. Nó xuất hiện hai lần trong `buildPieData`: một
cho `labels` (đi vào JSON của Chart.js) và một cho `legend`.

**Cạm bẫy khi grep:** tìm chuỗi tiếng Việt bằng dấu thanh sẽ **bỏ sót** các
nhãn không dấu. Ít nhất `Chi` (expense) và `Thu` (income) xuất hiện ở
`categories.html:11,17,74,75`, `transaction_row.html:40,44`,
`dashboard.html:75,76`. Phải rà thủ công từng template, không tin vào một
lệnh grep theo dấu.

## 6. Tham số URL `?thang=` → `?month=`

3 handler (`report_handlers.go:29`, `transaction_handlers.go:57,69`) và 4 chỗ
trong template (`dashboard.html`, `transactions.html`). Tham số này hiện trên
thanh địa chỉ vì dropdown chọn tháng dùng `hx-push-url="true"`.

Không làm redirect tương thích ngược cho link cũ: URL cũ sẽ mất filter và rơi
về tháng hiện tại. Chấp nhận được với app cá nhân chưa chia sẻ link ra ngoài.

## 7. Kiểm thử

Test hiện có đang assert chuỗi tiếng Việt (`"Ăn uống"`, `"Tháng 8"`,
`"50.000₫"`) ở `auth_handlers_test.go`, `category_handlers_test.go`,
`transaction_handlers_test.go`, `report_handlers_test.go`,
`report_handlers_internal_test.go`, `smoke_test.go`, `migrate_test.go`. Cập
nhật theo giá trị mới.

Test mới:

- `i18n.CategoryName`: slug hợp lệ → bản dịch; slug rỗng → `name` gốc; slug
  lạ → `name` gốc.
- `buildPieData`: breakdown có hàng slug `food_drink` → legend hiện
  `Food & Drink`; hơn 5 danh mục → hàng gộp hiện `Other` và trùng khớp với
  nhãn tương ứng trong `PieLabelsJSON`.
- Migration `000008`: chạy up trên database đã có hàng `Thu nhập khác` được
  một giao dịch tham chiếu → hàng vẫn còn, có slug `other_income`, giao dịch
  không mồ côi. Chạy down → `name` về lại tiếng Việt, cột slug biến mất.
- `formatThousands` / `dateFull` / `dateShort` với giá trị biên (0, số 3 chữ
  số, ngày đầu tháng).

**Lưu ý vận hành:** test trong `internal/handlers` đọc `TEST_DATABASE_URL`
trực tiếp và **không** tự chạy migration. Phải áp migration `000008` vào
database đó trước (chạy `cmd/server` trỏ vào nó là đủ), nếu không mọi test sẽ
fail vì thiếu cột `slug`.

## 8. Regenerate sqlc — có bẫy đã biết

Sửa `categories.sql` bắt buộc phải chạy `sqlc generate`, và việc đó kéo theo
một vấn đề đã ghi nhận: `internal/sqlcgen` hiện lệch với output của sqlc
v1.31.1. Regenerate sẽ đổi `GetCategoryWithTransactionCountParams.UserID` và
`ListCategoriesWithTransactionCounts` từ `pgtype.Int8` sang `int64`, làm
`internal/handlers/category_handlers.go` không compile ở 3 chỗ gọi đang
truyền `pgInt64(userID)`.

Cách né thông thường (`git checkout` lại file sau khi generate) **không dùng
được lần này**: `categories.sql.go` là file cần thay đổi ở mục 3, và
`transactions.sql.go` là file cần thay đổi ở mục 2(b) — tức là cả hai file
vẫn hay được khôi phục lần này đều phải giữ. Nên phải sửa 3 chỗ gọi đó sang
`int64` thuần, và việc này đi **commit riêng đầu tiên**, trước phần dịch, để
một thay đổi hạ tầng không lẫn vào một thay đổi nội dung.

Ngoài ra `slug` được thêm vào bảng khiến `SELECT *` trong `categories.sql`
sinh thêm field `Slug pgtype.Text` cho `sqlcgen.Category` — đúng như mong
muốn, không phải drift.

## 9. Tài liệu

- `SPEC.md`: thêm mục ghi rõ quy ước tiếng Anh (dấu phẩy, tháng viết chữ) đã
  thay quy ước Việt ở mục 1, giữ nguyên phần còn lại.
- `CLAUDE.md`: cập nhật đoạn mô tả helper format và đoạn mô tả danh mục mặc
  định (nay có slug, tên tiếng Anh).

## 10. Thứ tự commit

Nhánh `feat/english-ui`, merge `--no-ff` vào master khi xong.

1. `chore:` sửa 3 chỗ gọi sang `int64` để hết lệch với sqlc (mục 8)
2. `feat:` migration 000008 + `i18n.CategoryName` + `catName` + query slug (mục 1-3)
3. `refactor:` định dạng tiền/ngày/nhãn tháng (mục 4)
4. `feat:` chuỗi trong template và handler + `?month=` + docs (mục 5, 6, 9)

Mỗi commit build được và test xanh độc lập. Commit 2 làm database và code
lệch nhau tạm thời với ai đang chạy bản cũ — chỉ ảnh hưởng môi trường dev, và
Render deploy theo master sau khi merge nên production chỉ thấy trạng thái
cuối.
