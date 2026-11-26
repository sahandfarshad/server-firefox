# Server Firefox

پنل تحت وب برای نصب Firefox روی سرورهای لینوکسی، با دسترسی از مرورگر (noVNC + Nginx + SSL).
روی سرور اصلی نصب می‌شود و از داخل پنل، با رمز SSH، روی سرورهای دیگر Firefox نصب می‌کند.

## نصب روی سرور اصلی

```bash
bash <(curl -fsSL https://github.com/sahandfarshad/server-firefox/releases/latest/download/install.sh)
```

بعد از نصب، منو با دستور `sf` باز می‌شود و آدرس پنل بالای منو است.

## افزودن سرور

در پنل: IP، پورت و رمز SSH، دامنه‌ای که به IP آن سرور اشاره می‌کند، و نام کاربری و رمز ورود به Firefox.
بعد از نصب، Firefox از `https://دامنه` با همان نام کاربری و رمز باز می‌شود؛ تمام‌صفحه و با اتصال خودکار.

سرور مقصد: Ubuntu 22.04 / 24.04 یا Debian 12، پورت‌های 80 و 443 باز.

## امنیت

- رمزهای SSH و Firefox روی سرور اصلی با AES-GCM رمزنگاری می‌شوند.
- اثر کلید SSH هر سرور در اولین اتصال ذخیره و بعد بررسی می‌شود.
- رمزها از طریق stdin به سرور مقصد می‌روند و روی آن ذخیره نمی‌شوند (فقط هش bcrypt برای Nginx).
- VNC و websockify فقط روی localhost؛ ورود از همان ابتدا با Basic Auth.

## Languages

The panel opens in English; pick another language from the globe button (saved in the browser). 15 languages: English, فارسی, العربية, اردو, Türkçe, Türkmençe, Русский, Беларуская, 中文, 조선어, မြန်မာ, Tiếng Việt, Español, አማርኛ, ትግርኛ.
Translations live in `i18n/`: edit `i18n/strings.py` or `i18n/ui.src.html`, then run `python3 i18n/build.py` to regenerate `ui.html`.
