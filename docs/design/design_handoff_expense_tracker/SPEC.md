# Đặc tả giao diện — Sổ chi tiêu cá nhân

Web app expense tracker cá nhân. Mỗi người một tài khoản riêng, dữ liệu độc lập, không có tính
năng chia tiền nhóm. 5 màn hình: Đăng nhập/Đăng ký, Giao dịch, Danh mục, Tổng quan, và thanh
điều hướng dùng chung.

Fidelity: **high-fidelity**. Các con số dưới đây là giá trị cuối cùng, không phải gợi ý.

File tham khảo trực quan: `Expense Tracker UI.dc.html` (mở trong browser). Đó là design
reference, không phải production code — không copy markup, chỉ đọc để đối chiếu bố cục.

---

## 1. Design tokens

### Màu

| Vai trò | Giá trị | Ghi chú |
|---|---|---|
| Nền app | `#FAF9F7` | nền ngoài của mọi trang |
| Nền surface | `#FFFFFF` | card, nav, input, dòng danh sách |
| Nền surface phụ | `#FCFBF9` | hàng tổng ở cuối bảng |
| Nền segmented/track | `#F4F3F0` | rãnh của segmented control |
| Viền card | `#E9E7E4` | |
| Viền input | `#E2E0DC` | |
| Đường kẻ trong danh sách | `#F1EFEC` | |
| Viền nav | `#EDEBE7` | |
| Chữ chính | `#1B1A18` | |
| Chữ phụ | `#57534E` | label, nav item không active |
| Chữ mờ | `#8A8781` | mô tả, caption |
| Chữ rất mờ | `#9C9891` | ngày trong dòng giao dịch, tick biểu đồ |
| Placeholder | `#A8A49D` | |
| Chữ trong empty state (số 0₫) | `#C6C2BB` | |
| **Accent** | `#BC5A29` (tối: `#E08A5A`) | nút chính, nav active, focus ring |
| Accent nền nhạt | `color-mix(in oklab, accent 10%, transparent)` | nền nav item active |
| Chi (expense) | `#B42318` (tối: `#F97066`) | số tiền âm, cột "Chi" |
| Thu (income) | `#2F7D5B` (tối: `#57C398`) | số tiền dương, cột "Thu" |

Accent là **màu duy nhất** mang tính thương hiệu. Không dùng gradient. Không dùng màu danh mục
làm nền của bất cứ thứ gì.

Bảng trên là bản sáng; giá trị trong ngoặc là bản tối. Nguồn thi hành là các CSS variable
`--c-*` trong `internal/web/templates/layout.html` — sửa màu ở đó, không sửa rải rác trong
template.

Palette màu danh mục (chỉ dùng cho chấm 8–10px và cho biểu đồ):

```
#D97757  cam đất      #5B8DEF  xanh dương   #8B7BD8  tím
#6BA292  xanh lục xám  #E0A82E  vàng        #D97AA0  hồng
#4FA871  xanh lá      #7CA65C  xanh olive   #A1A1AA  xám
```

Người dùng chọn màu từ đúng 8 ô này (bảng chọn màu trong form thêm danh mục), không có
color picker tự do.

### Typography

- **Giao diện**: `Be Vietnam Pro`, weight 400/500/600/700. Fallback `system-ui, sans-serif`.
- **Số**: `JetBrains Mono`, weight 400/500/600, luôn kèm `font-variant-numeric: tabular-nums`.
  Dùng cho: mọi số tiền, ngày dạng `11/08/2026`, tick trục biểu đồ, phần trăm.
- `text-wrap: pretty` cho các đoạn mô tả.

Thang cỡ chữ (desktop → mobile):

| Dùng cho | Desktop | Mobile |
|---|---|---|
| Số tiền lớn (thẻ Tổng quan) | 32px / 600 / `-0.02em` | 27px / 600 |
| Tiêu đề trang | 18px / 600 | 19px / 600 / `-0.01em` |
| Tiêu đề card | 13–14px / 600 | 13px / 600 |
| Số tiền trong dòng giao dịch | 14px / 500 mono | 14px / 500 mono |
| Nội dung dòng, nút | 13px / 400–600 | 14–15px |
| Label form | 12px / 500 | 12px / 500 |
| Label form dạng uppercase (form thêm nhanh) | 11px / 600 / `0.04em` / uppercase | — |
| Caption, chữ mờ | 12px | 12px |
| Nhãn "Mặc định", ngày | 11–12px | 11–12px |

### Spacing, radius, shadow

- Radius: `8px` input/nút desktop · `10px` input/nút mobile · `12px` card · `999px` pill/chấm ·
  `6px` item bên trong segmented control · `5px` badge nhỏ.
- Gap chuẩn: `6px` (label→input), `10px`, `12px`, `16px`, `18px` (giữa các card), `20–28px` (giữa khối).
- Padding card: `16–22px` desktop, `16px` mobile.
- Chiều cao: input/nút desktop `36–38px`; input mobile `48px`; nút chính mobile `50px`;
  nav top `54px`; vùng chạm bottom bar `56px` (padding `10px 0 22px` để chừa safe-area).
- Shadow: gần như không dùng. Chỉ `0 1px 2px rgba(0,0,0,0.06)` cho ô đang chọn trong segmented
  control, và `0 8px 20px -6px accent/60%` cho FAB mobile.
- Focus: viền đổi sang accent + `box-shadow: 0 0 0 3px accent/12%`. Không dùng outline mặc định.

### Breakpoint

Một breakpoint duy nhất: **768px**. Dưới đó là layout mobile (bottom bar, một cột, bottom sheet),
từ đó lên là desktop (top nav, nội dung căn giữa rộng tối đa 880px).

### Format tiền

Integer đơn vị đồng → `50.000₫`. Dấu chấm phân cách nghìn, không phần thập phân, ký hiệu `₫`
liền sau số. Dấu: chi hiển thị `-85.000₫`, thu hiển thị `+18.000.000₫`. Số dư có thể âm hoặc dương,
luôn có dấu.

Ngày: `11/08/2026` trong form và cột ngày desktop; `11/08` trong dòng giao dịch.

> **Cập nhật 2026-08-16 — giao diện chuyển sang tiếng Anh.** Toàn bộ mục
> "Format tiền" ở trên mô tả quy ước Việt và **không còn đúng với code**. Quy
> ước hiện hành:
>
> - Tiền: dấu **phẩy** phân cách nghìn — `50,000₫`, `-85,000₫`, `+18,500,000₫`.
>   Ký hiệu `₫` vẫn liền sau số, vẫn không có phần thập phân.
> - Ngày: `11 Aug 2026` trong form và cột ngày desktop, `11 Aug` trong dòng
>   giao dịch. Tháng viết chữ để `11/08` không bị đọc thành hai ngày khác nhau
>   tuỳ người đọc là Anh hay Mỹ.
> - Nhãn tháng: `August 2026`; trục biểu đồ cột: `Aug`.
> - Dòng so sánh: `Last month 12,150,000₫ · down 11%`, bản mobile
>   `Down 11% vs last month`.
>
> Mọi chuỗi tiếng Việt còn lại trong tài liệu này là bản gốc của thiết kế, giữ
> nguyên làm tham chiếu; bản dịch tiếng Anh nằm trong template. Xem
> `docs/superpowers/specs/2026-08-16-english-ui-design.md`.

---

## 2. Màn hình: Đăng nhập / Đăng ký

**Mục đích**: vào app. Không có trang marketing, không hero, không hình minh hoạ.

**Bố cục desktop**: một cột căn giữa cả ngang lẫn dọc trên nền `#FAF9F7`. Cột rộng `380px`,
padding dọc `72px`.

Từ trên xuống:
1. Logo: ô vuông `30×30`, radius `9px`, nền accent. Không chữ trong logo.
2. Tên app `Sổ chi tiêu` — 17px/600.
3. Dòng phụ `Ghi lại thu chi hằng ngày của bạn` — 13px, `#8A8781`.
4. Card trắng, viền `#E9E7E4`, radius `12px`, padding `22px`, gap `16px`:
   - Segmented control 2 tab `Đăng nhập` / `Đăng ký`, full width, nền rãnh `#F4F3F0`,
     padding rãnh `3px`, tab đang chọn nền trắng + shadow nhẹ + 600.
   - Field Email: label 12px/500 `#57534E`, input cao 38px, placeholder `ban@email.com`.
   - Field Mật khẩu: label bên trái, link `Quên mật khẩu?` 12px màu accent căn phải cùng hàng
     baseline với label. Input `type=password`.
   - Nút chính full width, cao 40px, nền accent, chữ trắng 13px/600, nhãn `Đăng nhập`.
5. Dòng dưới card, căn giữa, 12px: `Chưa có tài khoản?` + link `Đăng ký` màu accent, 500.

**Tab Đăng ký**: cùng card, thêm field `Tên` (trên Email) và `Nhập lại mật khẩu` (dưới Mật khẩu),
bỏ link `Quên mật khẩu?`, nút thành `Tạo tài khoản`. Đổi tab bằng htmx swap phần thân card
(giữ nguyên logo và tab bar) — không điều hướng trang.

**Lỗi**: một khối trên nút chính — nền `#FEF2F2`, viền `#FECACA`, chữ màu Chi 12px, radius 8px.
Nội dung: `Email hoặc mật khẩu không đúng.` / `Email này đã được dùng.` /
`Mật khẩu phải có ít nhất 8 ký tự.` Server trả về fragment card kèm lỗi, giữ nguyên email đã nhập,
xóa mật khẩu.

**Mobile**: bỏ card (không viền, không nền trắng riêng), nội dung chiếm hết bề ngang, padding
`48px 20px`. Logo `34×34` căn trái, tiêu đề `Đăng nhập` 22px/600 căn trái thay vì căn giữa.
Input cao `48px`, cỡ chữ `15px` (tránh iOS zoom khi focus). Nút cao `50px`.

**Empty state**: không có.

---

## 3. Màn hình: Giao dịch (màn hình chính)

**Mục đích**: nhập giao dịch nhanh và xem lại giao dịch trong tháng.

**Bố cục desktop**: top nav 54px (mục 6), rồi vùng nội dung nền `#FAF9F7` padding `28px 24px 36px`,
bên trong là một cột rộng `880px` căn giữa, gap `18px`.

### 3.1 Form thêm nhanh

Card trắng padding `16px`. Một hàng flex `align-items: flex-end`, gap `10px`. Mỗi field là một
cột nhỏ: label uppercase 11px/600 `#8A8781` phía trên, control cao 36px phía dưới.

| Field | Width | Control |
|---|---|---|
| Loại | auto | segmented `Chi` / `Thu`, mặc định `Chi` |
| Danh mục | 168px | select, hiển thị chấm màu 8px + tên, mũi tên `▾` màu `#A8A49D` căn phải |
| Số tiền | 150px | input số, mono 14px/500, hiển thị đã format khi blur |
| Ngày | 128px | date input, mono 13px, mặc định hôm nay |
| Ghi chú | flex:1 | text input, placeholder `Không bắt buộc` |
| — | auto | nút `Thêm`, cao 36px, padding ngang 18px, nền accent, chữ trắng 13px/600 |

Hành vi:
- Danh sách danh mục trong select **lọc theo Loại đang chọn**. Đổi Loại → `hx-get` nạp lại
  `<select>` (swap `outerHTML`).
- Submit `hx-post` → server trả về `<tr>`/row mới, `hx-swap="afterbegin"` vào đầu danh sách;
  đồng thời cập nhật hàng tổng và hai số Chi/Thu ở thanh lọc (dùng `hx-swap-oob`).
- Sau khi thêm thành công: reset Số tiền và Ghi chú, **giữ nguyên** Loại, Danh mục, Ngày
  (người dùng thường nhập liền nhiều giao dịch cùng ngày). Focus trở lại ô Số tiền.
- Lỗi validate: viền input đỏ `#C2410C` + dòng lỗi 12px đỏ ngay dưới input đó, không dùng alert.

### 3.2 Thanh lọc tháng

Hàng flex `space-between`, ngay dưới form:
- Trái: nút chọn tháng cao 32px, viền `#E2E0DC`, nền trắng, radius 8px, chữ 13px/500
  `Tháng 8, 2026` + `▾`. Bên cạnh là `8 giao dịch` 12px `#8A8781`.
- Phải: `Chi 10.800.000₫` (số màu `#C2410C`, mono 500) và `Thu 18.500.000₫`
  (số màu `#2F7D5B`), nhãn `Chi`/`Thu` 13px `#8A8781`, gap 18px.

Chọn tháng: dropdown liệt kê các tháng có dữ liệu (mới nhất trên cùng) + `Tháng này`.
`hx-get` thay toàn bộ khối danh sách + hai số tổng. Cập nhật query param `?thang=2026-08`
bằng `hx-push-url` để reload/back hoạt động đúng.

### 3.3 Danh sách giao dịch

Card trắng radius 12px, `overflow: hidden`. Mỗi dòng padding `13px 16px`, border-bottom
`#F1EFEC`, flex gap `16px`, cột cố định để số liệu thẳng hàng:

| Cột | Width | Nội dung |
|---|---|---|
| Ngày | 46px | `11/08` mono 12px `#9C9891` |
| Danh mục | 150px | chấm 8px + tên 13px/500 |
| Ghi chú | flex:1 | 13px `#8A8781`, `text-overflow: ellipsis` một dòng |
| Hành động | auto | `Sửa` · `Xóa` — 12px `#8A8781`, viền `#EAE8E4`, radius 6px, padding `3px 7px`; **opacity 0 mặc định, hiện khi hover dòng** (mobile: luôn ẩn, thay bằng swipe) |
| Số tiền | 132px | mono 14px/500 căn phải, `-` màu `#C2410C` / `+` màu `#2F7D5B` |

Hover dòng: nền `#FCFBF9`.

Hàng cuối (không border-bottom), nền `#FCFBF9`, padding `13px 16px`:
`Còn lại tháng này` 13px/600 bên trái, số mono 15px/600 căn phải trong cùng cột 132px.
Số dương màu `#1B1A18`, số âm màu `#C2410C`.

**Sửa**: `hx-get` trả về chính dòng đó ở dạng các input inline (cùng cột, cùng chiều rộng),
hai nút `Lưu` / `Hủy` thay cho `Sửa`/`Xóa`. Không mở modal trên desktop.

**Xóa**: click `Xóa` → dòng đó chuyển thành thanh xác nhận inline
`Xóa giao dịch này?  [Xóa]  [Hủy]`, nền `#FEF7F5`. Xác nhận → `hx-delete`, dòng biến mất,
tổng cập nhật. Không dùng `confirm()` của browser.

**Empty state** (chưa có giao dịch trong tháng đang chọn): card trắng, padding `44px 28px`,
nội dung căn giữa:
- Tiêu đề 15px/600: `Chưa có giao dịch nào trong tháng 8`
- Mô tả 13px `#8A8781`, max-width 300px:
  `Thêm giao dịch đầu tiên bằng form phía trên, hoặc chọn tháng khác để xem lại.`
- Nút accent `Thêm giao dịch` (focus vào ô Số tiền của form phía trên; trên mobile mở bottom sheet).
- Không hình minh hoạ, không viền đứt. Hàng tổng vẫn hiển thị `0₫`.

### 3.4 Mobile

- Header: tiêu đề `Giao dịch` 19px/600 bên trái, nút chọn tháng `Th 8, 2026 ▾` cao 32px bên phải.
- Hai thẻ tổng nhỏ nằm ngang, mỗi thẻ flex:1, padding `10px 12px`, radius 10px:
  nhãn `Chi`/`Thu` 11px `#8A8781`, số mono 15px/600 màu tương ứng.
- Dòng giao dịch 2 tầng, padding `12px 13px`:
  chấm màu 8px bên trái → khối giữa (tên danh mục 14px/500; dòng dưới `11/08 · Cà phê với Hà`
  12px `#9C9891`, ellipsis) → số tiền mono 14px/500 căn phải.
- Sửa/xóa: vuốt trái để lộ hai nút; hoặc nhấn giữ mở action sheet. Không hiện nút thường trực.
- FAB: pill nền accent, cao 48px, padding ngang 20px, nhãn `＋ Thêm`, đặt `right: 16px`,
  cách bottom bar `~74px`, shadow `0 8px 20px -6px accent/60%`.
- FAB mở **bottom sheet** (radius trên 20px, padding `12px 18px 28px`, backdrop
  `rgba(27,26,24,0.32)`, có tay cầm 38×4px `#E2E0DC` căn giữa):
  1. Tiêu đề `Thêm giao dịch` 17px/600
  2. Segmented `Chi`/`Thu` full width, mỗi ô cao 40px, chữ 14px
  3. Field Số tiền — ô cao **56px**, mono **24px/600**, viền accent + focus ring. Đây là phần
     nổi bật nhất của sheet; bàn phím số bật sẵn (`inputmode="numeric"`).
  4. Danh mục dạng **chip cuộn ngang / wrap**: pill padding `9px 13px`, radius 999px,
     chấm màu 8px + tên 13px. Chip đang chọn: viền accent + nền accent 8%.
  5. Hàng cuối: ngày (mono 14px) và ghi chú, mỗi ô flex:1 cao 48px.
  6. Nút `Lưu giao dịch` full width cao 50px nền accent.

---

## 4. Màn hình: Danh mục

**Mục đích**: quản lý danh mục thu/chi.

**Bố cục desktop**: cột 880px, chia hai: danh sách `flex:1` bên trái, form thêm `300px` bên phải
(`position: sticky; top: 78px`), gap `20px`.

### 4.1 Danh sách

Chia hai nhóm, mỗi nhóm có heading uppercase 11px/600 `#8A8781` (`Chi`, `Thu`) rồi một card
trắng. Mỗi dòng padding `12px 14px`, border-bottom `#F1EFEC`, flex gap `12px`:

chấm màu **10px** → tên 13px/500 → badge loại (11px `#A8A49D`, viền `#EDEBE7`, radius 5px,
padding `1px 6px`: `Mặc định` hoặc `Tự tạo`) → (đẩy phải) số lượng `14 giao dịch` mono 12px
`#9C9891` → hành động 12px `#8A8781`.

Hành động khác nhau theo loại danh mục:
- Danh mục **mặc định**: chỉ `Đổi màu`. **Không có nút Xóa.** Không đổi được tên, không đổi
  được loại thu/chi. Handler phải từ chối request xóa/đổi tên danh mục mặc định (403), không chỉ
  ẩn nút ở UI.
- Danh mục **tự tạo**: `Sửa` · `Xóa`.

`Đổi màu` mở popover nhỏ ngay dưới dòng đó chứa đúng bảng 8 màu; chọn màu → `hx-patch`,
chấm cập nhật ngay.

**Xóa danh mục tự tạo đang có giao dịch**: dialog nhỏ (không phải `confirm()`):
tiêu đề `Xóa danh mục "Bán đồ cũ"?`, mô tả
`Danh mục này đang có 2 giao dịch. Các giao dịch sẽ được chuyển sang "Khác".`,
hai nút `Hủy` (viền) và `Xóa` (nền `#C2410C`). Nếu chưa có giao dịch thì mô tả rút gọn thành
`Hành động này không thể hoàn tác.`

### 4.2 Form thêm

Card trắng 300px, padding 18px, gap 15px:
1. Tiêu đề `Thêm danh mục` 14px/600
2. Field `Tên danh mục`, input 36px, placeholder `VD: Học phí`
3. Field `Loại`: segmented `Chi`/`Thu` full width
4. Field `Màu`: 8 ô chọn màu, mỗi ô `26×26`, radius 999px, chứa chấm màu `14×14` ở giữa;
   ô đang chọn có `border: 2px solid accent`, ô còn lại `border: 2px solid transparent`.
   Wrap, gap 8px.
5. Nút `Thêm danh mục` full width cao 38px nền accent
6. Ghi chú 12px `#A8A49D`: `Danh mục mặc định không thể xóa, chỉ đổi được màu.`

Submit `hx-post` → prepend dòng mới vào nhóm tương ứng, reset form, giữ nguyên Loại đang chọn.
Trùng tên trong cùng loại → lỗi 12px đỏ dưới input tên: `Đã có danh mục tên này.`

### 4.3 Danh mục mặc định (seed khi tạo tài khoản)

Chi: `Ăn uống` `#D97757` · `Đi lại` `#5B8DEF` · `Mua sắm` `#8B7BD8` · `Hóa đơn` `#6BA292` ·
`Giải trí` `#E0A82E` · `Sức khỏe` `#D97AA0` · `Khác` `#A1A1AA`
Thu: `Lương` `#4FA871` · `Thưởng` `#7CA65C`

Tổng 9 danh mục. `Khác` là nơi nhận giao dịch khi một danh mục tự tạo bị xóa, nên cũng không
xóa được.

**Empty state** (chưa có danh mục tự tạo — danh mục mặc định luôn tồn tại nên danh sách không
bao giờ rỗng hoàn toàn): thay vì card rỗng, hiện một dòng nhắc dưới nhóm cuối, 13px `#8A8781`,
căn giữa, padding 44px:
- `Bạn chưa tạo danh mục riêng`
- `9 danh mục mặc định đã sẵn sàng. Tạo thêm khi bạn cần theo dõi khoản chi riêng.`
- Nút phụ (viền `#E2E0DC`, chữ đen) `Thêm danh mục`.

### 4.4 Mobile

Một cột. Header: `Danh mục` 19px/600 bên trái, nút tròn `34×34` nền accent chữ `＋` bên phải →
mở bottom sheet chứa form thêm (cùng nội dung, input cao 48px).
Dòng danh mục: chấm 10px + tên 14px/500 + (đẩy phải) badge `Mặc định`/`Tự tạo` 12px `#A8A49D`.
Bỏ cột số lượng giao dịch. Sửa/xóa qua nhấn giữ → action sheet
(`Đổi màu` / `Đổi tên` / `Xóa`, ẩn mục không hợp lệ với danh mục mặc định).

---

## 5. Màn hình: Tổng quan

**Mục đích**: xem tổng thu/chi tháng và cơ cấu chi tiêu.

**Bố cục desktop**: cột 880px, gap 18px.

1. **Hàng tiêu đề**: `Tháng 8, 2026` 18px/600 bên trái; nút chọn tháng (giống trang Giao dịch)
   bên phải.
2. **Hai thẻ số liệu**, cạnh nhau, mỗi thẻ `flex:1`, card trắng padding 20px, gap 8px:
   - nhãn 12px/500 `#8A8781`: `Tổng chi tháng này` / `Tổng thu tháng này`
   - số mono **32px/600** `letter-spacing: -0.02em`, màu `#C2410C` / `#2F7D5B`
   - dòng so sánh 12px `#8A8781`: `Tháng trước 12.150.000₫ · giảm 11%`
     (từ `tăng`/`giảm`; nếu tháng trước không có dữ liệu → `Chưa có dữ liệu tháng trước`)
3. **Hàng hai biểu đồ**, gap 16px, cao bằng nhau:

   **Biểu đồ tròn — `Chi theo danh mục`** (card `440px`, padding 18px):
   - Chart.js `doughnut`, `cutout: '62%'`, canvas `150×150` (render 2× cho retina),
     `borderWidth: 2`, `borderColor: '#fff'`, tắt legend và tooltip mặc định.
   - Legend tự dựng bên phải, gap 9px, mỗi dòng: chấm 8px → tên 12px `#57534E` (flex:1) →
     `%` mono 12px `#9C9891` width 34px căn phải → số tiền mono 12px/500 width 86px căn phải.
   - Sắp xếp giảm dần theo số tiền. Chỉ chi (không gộp thu). Nếu > 6 danh mục: hiện 6 lớn nhất,
     phần còn lại gộp thành `Khác` màu `#A1A1AA`.
   - Hover một dòng legend → làm nổi cung tương ứng (`setActiveElements`).

   **Biểu đồ cột — `4 tháng gần nhất`** (card `flex:1`, padding 18px):
   - Header card: tiêu đề 13px/600 bên trái; legend nhỏ bên phải — ô vuông 8px radius 2px
     `#C2410C` + `Chi`, `#2F7D5B` + `Thu`, chữ 11px `#8A8781`.
   - Chart.js `bar`, hai dataset nhóm cạnh nhau, `borderRadius: 3`, `barPercentage: 0.62`,
     `categoryPercentage: 0.6`, cao ~158px.
   - Trục X: nhãn `Th 5`…`Th 8`, mono, `#9C9891`, bỏ grid và border.
   - Trục Y: `beginAtZero`, grid `#F1EFEC`, tick mono `#B5B1AA`, `maxTicksLimit: 4`,
     format rút gọn `12tr` (triệu). Bỏ border trục.
   - Tắt animation (`animation: false`) để htmx swap không gây nhảy layout.

**Truyền dữ liệu cho Chart.js**: render một `<script type="application/json" id="chart-data">`
chứa `{pie: {labels, values, colors}, bars: {labels, chi, thu}}`, JS đọc và khởi tạo.
Không nhúng biểu thức Go vào giữa code JS. Sau mỗi htmx swap, destroy chart cũ trước khi tạo mới
(`chart.destroy()`) để tránh leak canvas.

**Empty state** (tháng chưa có giao dịch nào):
- Hai thẻ số liệu **vẫn hiển thị**, số là `0₫` màu `#C6C2BB`, dòng so sánh ẩn.
- Vùng hai biểu đồ thay bằng một card trắng padding `44px 28px` căn giữa:
  - `Chưa đủ dữ liệu để vẽ biểu đồ` 15px/600
  - `Hai thẻ số liệu vẫn hiển thị 0₫. Biểu đồ xuất hiện sau giao dịch đầu tiên.` 13px `#8A8781`
  - Link accent `Thêm giao dịch` → trang Giao dịch.
- Biểu đồ cột: nếu có ít nhất 1 tháng có dữ liệu thì vẫn vẽ, các tháng rỗng là cột 0.

**Mobile**: một cột, cuộn dọc, gap 12px, padding ngang 14px.
- Header: `Tổng quan` 19px/600 + nút `Th 8, 2026 ▾`.
- Hai thẻ số liệu **xếp dọc**, số 27px, dòng so sánh rút ngắn thành `Giảm 11% so với tháng trước`.
- Card biểu đồ tròn: canvas `108×108` bên trái, legend bên phải nhưng **bỏ cột số tiền**,
  chỉ còn chấm + tên + `%`. Chỉ hiện 4 danh mục lớn nhất, còn lại gộp `Khác`.
- Card biểu đồ cột: canvas full width, cao ~118px, cỡ chữ tick nhỏ hơn một bậc.

---

## 6. Thanh điều hướng

Xuất hiện trên mọi trang sau khi đăng nhập. 4 đích: `Tổng quan` / `Giao dịch` / `Danh mục` /
`Đăng xuất`. **Không dùng icon** — chỉ chữ.

### Desktop (≥768px)

Thanh ngang cao `54px`, nền trắng, border-bottom `#EDEBE7`, `position: sticky; top: 0`,
padding ngang 24px, gap 26px:
- Trái: logo `18×18` radius 5px nền accent + `Sổ chi tiêu` 14px/600.
- Giữa: 3 link, mỗi link padding `6px 11px`, radius 7px, 13px.
  - Active: chữ accent 600 + nền `accent 10%`
  - Hover: nền `#F4F3F0`, chữ `#1B1A18`
  - Mặc định: chữ `#6B6862`
- Phải (đẩy bằng `margin-left: auto`): avatar tròn `24×24` nền `#EDEBE7` chứa chữ cái đầu
  11px/600 `#57534E`, tên người dùng 13px `#57534E`, rồi `Đăng xuất` 13px `#8A8781`
  (hover `#1B1A18`).

### Mobile (<768px)

Bottom bar `position: fixed; bottom: 0`, nền trắng, border-top `#EDEBE7`,
padding `10px 0 22px` (safe-area), 4 ô `flex: 1`. Mỗi ô: chấm `5×5` radius 999px phía trên
(accent khi active, `transparent` khi không) + nhãn 11px. Active: chữ accent 600.
Không active: `#8A8781`. Vùng chạm tối thiểu `56px` cao.

Nội dung trang phải có `padding-bottom` ≈ 76px để không bị bottom bar che.

### Đăng xuất

Là **form POST** kèm CSRF token, không phải link GET. Trên mobile, nhấn ô `Đăng xuất` mở dialog
nhỏ: `Đăng xuất khỏi Sổ chi tiêu?` + `Hủy` / `Đăng xuất`. Trên desktop submit trực tiếp.

---

## 7. Trạng thái khác

- **Loading**: htmx request > 300ms → nút đang submit chuyển sang `opacity: 0.6` +
  `pointer-events: none`, nhãn giữ nguyên (không đổi thành spinner). Danh sách đang nạp lại:
  cả khối `opacity: 0.5`. Không dùng skeleton.
- **Lỗi mạng**: banner trên đầu vùng nội dung, nền `#FEF2F2`, viền `#FECACA`, chữ `#C2410C` 13px:
  `Không lưu được. Kiểm tra kết nối và thử lại.` + nút `Thử lại`.
- **Thành công**: không toast. Dữ liệu xuất hiện trong danh sách là đủ phản hồi.

## 8. Ràng buộc dữ liệu

- Số tiền: integer, đơn vị đồng, > 0. Loại (`chi`/`thu`) quyết định dấu khi hiển thị.
- Giao dịch bắt buộc: loại, danh mục, số tiền, ngày. Ghi chú tùy chọn, tối đa 200 ký tự.
- Loại của giao dịch phải khớp loại của danh mục được chọn.
- Danh mục thuộc về một user; tên duy nhất trong phạm vi (user, loại).
- Mọi truy vấn scope theo `user_id`. Kiểm tra quyền sở hữu ở handler cho mọi sửa/xóa.

## 9. Assets

Không có ảnh, không có icon library. Logo là ô vuông màu accent. Font tải từ Google Fonts:
`Be Vietnam Pro` (400,500,600,700) và `JetBrains Mono` (400,500,600) — nên self-host nếu repo
đã có cơ chế đó.
