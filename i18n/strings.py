# Server Firefox UI strings. Shared panel strings come from base_strings.py
# (same as Server Desktop); Firefox-specific ones are below.
from base_strings import KEYS as BK, T as BT
from langs import LANGS

REUSE = ["lang","user","pass","login","wrong","add","logout","empty","add_first","new_server","edit_title","name_opt",
"name_ph","ip","ssh_port","ssh_user","ssh_pass","clear_key","save_install","save_only","cancel","hint_edit",
"st_new","st_installing","st_installed","st_failed","st_removed","run_install","run_status","run_restart","run_uninstall",
"status","reinstall","edit","log","delete","no_job","c_reinstall"]

NEW = ["domain","web_user","web_pass","resolution","hint_new","open_ff","install","restart","c_uninstall","c_remove",
"e_host","e_domain","e_user","e_pass","e_sshpass"]

N = {}
N["en"] = ["Domain (must point to this server's IP)","Firefox login username","Firefox login password (min. 8)","Screen resolution",
"Before installing, point the domain's A record to the server IP and open ports 80 and 443. OS: Ubuntu 22.04 / 24.04 or Debian 12.",
"Open Firefox ↗","Install Firefox","Restart Firefox","Remove Firefox from {name}?",
"{name}\n1 = also remove Firefox from the server, then delete it from the list\n2 = only delete it from the list",
"Invalid server IP or host","Invalid domain (example: fire.example.com)",
"Firefox username: 3 to 32 letters, digits, dot, dash or underscore","Firefox password must be at least 8 characters","SSH password is required"]
N["fa"] = ["دامنه (باید به IP همین سرور اشاره کند)","نام کاربری ورود به Firefox","رمز ورود به Firefox (حداقل ۸)","رزولوشن صفحه",
"قبل از نصب، رکورد A دامنه را روی IP سرور بگذار و پورت‌های 80 و 443 باز باشند. سیستم‌عامل: Ubuntu 22.04 / 24.04 یا Debian 12.",
"باز کردن Firefox ↗","نصب Firefox","ری‌استارت Firefox","Firefox از {name} حذف شود؟",
"{name}\n۱ = Firefox را از خود سرور هم پاک کن، بعد از لیست حذف کن\n۲ = فقط از لیست حذف کن",
"IP یا آدرس سرور نامعتبر است","دامنه نامعتبر است (مثال: fire.example.com)",
"نام کاربری Firefox: ۳ تا ۳۲ حرف، عدد، نقطه، خط تیره یا زیرخط","رمز Firefox باید حداقل ۸ کاراکتر باشد","رمز SSH لازم است"]
N["ar"] = ["النطاق (يجب أن يشير إلى IP هذا الخادم)","اسم مستخدم دخول Firefox","كلمة مرور دخول Firefox (8 على الأقل)","دقة الشاشة",
"قبل التثبيت، وجّه سجل A للنطاق إلى IP الخادم وافتح المنفذين 80 و443. النظام: Ubuntu 22.04 / 24.04 أو Debian 12.",
"فتح Firefox ↗","تثبيت Firefox","إعادة تشغيل Firefox","حذف Firefox من {name}؟",
"{name}\n1 = احذف Firefox من الخادم أيضاً، ثم من القائمة\n2 = احذفه من القائمة فقط",
"عنوان IP أو اسم الخادم غير صالح","نطاق غير صالح (مثال: fire.example.com)",
"اسم مستخدم Firefox: من 3 إلى 32 حرفاً أو رقماً أو نقطة أو شرطة أو شرطة سفلية","يجب أن تكون كلمة مرور Firefox من 8 أحرف على الأقل","كلمة مرور SSH مطلوبة"]
N["ur"] = ["ڈومین (اسی سرور کے IP کی طرف اشارہ کرے)","Firefox لاگ اِن صارف نام","Firefox لاگ اِن پاس ورڈ (کم از کم 8)","اسکرین ریزولوشن",
"انسٹال سے پہلے ڈومین کا A ریکارڈ سرور IP پر لگائیں اور پورٹ 80 اور 443 کھولیں۔ OS: Ubuntu 22.04 / 24.04 یا Debian 12۔",
"Firefox کھولیں ↗","Firefox انسٹال کریں","Firefox ری اسٹارٹ","{name} سے Firefox ہٹائیں؟",
"{name}\n1 = سرور سے Firefox بھی ہٹائیں، پھر فہرست سے حذف کریں\n2 = صرف فہرست سے حذف کریں",
"سرور IP یا ہوسٹ غلط ہے","ڈومین غلط ہے (مثال: fire.example.com)",
"Firefox صارف نام: 3 سے 32 حروف، ہندسے، نقطہ، ڈیش یا انڈر اسکور","Firefox پاس ورڈ کم از کم 8 حروف کا ہو","SSH پاس ورڈ ضروری ہے"]
N["tr"] = ["Alan adı (bu sunucunun IP'sini göstermeli)","Firefox giriş kullanıcı adı","Firefox giriş parolası (en az 8)","Ekran çözünürlüğü",
"Kurmadan önce alan adının A kaydını sunucu IP'sine yönlendirin ve 80 ile 443 portlarını açın. İS: Ubuntu 22.04 / 24.04 veya Debian 12.",
"Firefox'u aç ↗","Firefox'u kur","Firefox'u yeniden başlat","Firefox {name} sunucusundan kaldırılsın mı?",
"{name}\n1 = Firefox'u sunucudan da kaldır, sonra listeden sil\n2 = yalnızca listeden sil",
"Geçersiz sunucu IP'si veya adı","Geçersiz alan adı (örnek: fire.example.com)",
"Firefox kullanıcı adı: 3-32 harf, rakam, nokta, tire veya alt çizgi","Firefox parolası en az 8 karakter olmalı","SSH parolası gerekli"]
N["tk"] = ["Domen (şu serweriň IP-sine görkezmeli)","Firefox giriş ulanyjy ady","Firefox giriş açar sözi (iň az 8)","Ekranyň çözgüdi",
"Gurnamazdan öň domeniň A ýazgysyny serweriň IP-sine görkeziň we 80 hem-de 443 portlaryny açyň. OS: Ubuntu 22.04 / 24.04 ýa-da Debian 12.",
"Firefox-y aç ↗","Firefox gurna","Firefox-y täzeden işlet","Firefox {name} serwerinden aýrylsynmy?",
"{name}\n1 = Firefox-y serwerden hem aýyr, soň sanawdan poz\n2 = diňe sanawdan poz",
"Serweriň IP-si ýa-da ady nädogry","Domen nädogry (mysal: fire.example.com)",
"Firefox ulanyjy ady: 3-den 32-ä çenli harp, san, nokat, defis ýa-da aşaky çyzyk","Firefox açar sözi iň az 8 nyşan bolmaly","SSH açar sözi hökmany"]
N["ru"] = ["Домен (должен указывать на IP этого сервера)","Логин для входа в Firefox","Пароль для входа в Firefox (мин. 8)","Разрешение экрана",
"Перед установкой направьте A-запись домена на IP сервера и откройте порты 80 и 443. ОС: Ubuntu 22.04 / 24.04 или Debian 12.",
"Открыть Firefox ↗","Установить Firefox","Перезапустить Firefox","Удалить Firefox с {name}?",
"{name}\n1 = удалить Firefox и с сервера, затем из списка\n2 = только удалить из списка",
"Неверный IP или адрес сервера","Неверный домен (пример: fire.example.com)",
"Логин Firefox: от 3 до 32 букв, цифр, точек, дефисов или подчёркиваний","Пароль Firefox должен быть не короче 8 символов","Нужен пароль SSH"]
N["be"] = ["Дамен (павінен паказваць на IP гэтага сервера)","Лагін для ўваходу ў Firefox","Пароль для ўваходу ў Firefox (мін. 8)","Раздзяляльнасць экрана",
"Перад усталяваннем накіруйце A-запіс дамена на IP сервера і адкрыйце парты 80 і 443. АС: Ubuntu 22.04 / 24.04 або Debian 12.",
"Адкрыць Firefox ↗","Усталяваць Firefox","Перазапусціць Firefox","Выдаліць Firefox з {name}?",
"{name}\n1 = выдаліць Firefox таксама з сервера, потым са спіса\n2 = толькі выдаліць са спіса",
"Няправільны IP або адрас сервера","Няправільны дамен (прыклад: fire.example.com)",
"Лагін Firefox: ад 3 да 32 літар, лічбаў, кропак, злучкоў або падкрэсленняў","Пароль Firefox павінен мець не менш за 8 сімвалаў","Патрэбны пароль SSH"]
N["zh"] = ["域名（必须指向此服务器的 IP）","Firefox 登录用户名","Firefox 登录密码（至少 8 位）","屏幕分辨率",
"安装前，请将域名的 A 记录指向服务器 IP，并开放 80 和 443 端口。系统：Ubuntu 22.04 / 24.04 或 Debian 12。",
"打开 Firefox ↗","安装 Firefox","重启 Firefox","从 {name} 移除 Firefox？",
"{name}\n1 = 同时从服务器移除 Firefox，然后从列表删除\n2 = 仅从列表删除",
"服务器 IP 或主机名无效","域名无效（例如：fire.example.com）",
"Firefox 用户名：3 到 32 个字母、数字、点、连字符或下划线","Firefox 密码至少 8 个字符","需要 SSH 密码"]
N["ko"] = ["도메인 (이 봉사기의 IP를 가리켜야 함)","Firefox 가입 사용자 이름","Firefox 가입 암호 (최소 8자)","화면 해상도",
"설치하기 전에 도메인의 A 기록을 봉사기 IP로 지정하고 80, 443 포구를 여십시오. 조작체계: Ubuntu 22.04 / 24.04 또는 Debian 12.",
"Firefox 열기 ↗","Firefox 설치","Firefox 재시동","{name}에서 Firefox를 제거하겠습니까?",
"{name}\n1 = 봉사기에서도 Firefox를 제거한 다음 목록에서 삭제\n2 = 목록에서만 삭제",
"봉사기 IP 또는 주소가 올바르지 않습니다","도메인이 올바르지 않습니다 (례: fire.example.com)",
"Firefox 사용자 이름: 3~32자의 글자, 수자, 점, 줄표 또는 밑줄","Firefox 암호는 최소 8자여야 합니다","SSH 암호가 필요합니다"]
N["my"] = ["ဒိုမိန်း (ဤဆာဗာ၏ IP သို့ ညွှန်ပြရမည်)","Firefox ဝင်ရောက်ရန် အသုံးပြုသူအမည်","Firefox ဝင်ရောက်ရန် စကားဝှက် (အနည်းဆုံး 8)","မျက်နှာပြင် ရီဆိုလူးရှင်း",
"မထည့်သွင်းမီ ဒိုမိန်း၏ A record ကို ဆာဗာ IP သို့ ညွှန်ပြီး ပို့တ် 80 နှင့် 443 ကို ဖွင့်ပါ။ OS: Ubuntu 22.04 / 24.04 သို့မဟုတ် Debian 12။",
"Firefox ဖွင့်ရန် ↗","Firefox ထည့်သွင်းရန်","Firefox ပြန်စရန်","{name} မှ Firefox ကို ဖယ်ရှားမလား။",
"{name}\n1 = ဆာဗာမှ Firefox ကိုပါ ဖယ်ရှားပြီး စာရင်းမှ ဖျက်ရန်\n2 = စာရင်းမှသာ ဖျက်ရန်",
"ဆာဗာ IP သို့မဟုတ် host မမှန်ပါ","ဒိုမိန်း မမှန်ပါ (ဥပမာ: fire.example.com)",
"Firefox အသုံးပြုသူအမည်: စာလုံး၊ ဂဏန်း၊ အစက်၊ တုံးတို သို့မဟုတ် underscore 3 မှ 32 လုံး","Firefox စကားဝှက်သည် အနည်းဆုံး စာလုံး 8 လုံး ရှိရမည်","SSH စကားဝှက် လိုအပ်သည်"]
N["vi"] = ["Tên miền (phải trỏ tới IP của máy chủ này)","Tên đăng nhập Firefox","Mật khẩu đăng nhập Firefox (tối thiểu 8)","Độ phân giải màn hình",
"Trước khi cài, hãy trỏ bản ghi A của tên miền tới IP máy chủ và mở cổng 80 và 443. HĐH: Ubuntu 22.04 / 24.04 hoặc Debian 12.",
"Mở Firefox ↗","Cài Firefox","Khởi động lại Firefox","Gỡ Firefox khỏi {name}?",
"{name}\n1 = gỡ cả Firefox khỏi máy chủ, rồi xóa khỏi danh sách\n2 = chỉ xóa khỏi danh sách",
"IP hoặc địa chỉ máy chủ không hợp lệ","Tên miền không hợp lệ (ví dụ: fire.example.com)",
"Tên người dùng Firefox: 3 đến 32 chữ cái, chữ số, dấu chấm, gạch ngang hoặc gạch dưới","Mật khẩu Firefox phải có ít nhất 8 ký tự","Cần mật khẩu SSH"]
N["es"] = ["Dominio (debe apuntar a la IP de este servidor)","Usuario de acceso a Firefox","Contraseña de acceso a Firefox (mín. 8)","Resolución de pantalla",
"Antes de instalar, apunta el registro A del dominio a la IP del servidor y abre los puertos 80 y 443. SO: Ubuntu 22.04 / 24.04 o Debian 12.",
"Abrir Firefox ↗","Instalar Firefox","Reiniciar Firefox","¿Quitar Firefox de {name}?",
"{name}\n1 = quitar también Firefox del servidor y luego borrarlo de la lista\n2 = solo borrarlo de la lista",
"IP o host del servidor no válido","Dominio no válido (ejemplo: fire.example.com)",
"Usuario de Firefox: de 3 a 32 letras, dígitos, punto, guion o guion bajo","La contraseña de Firefox debe tener al menos 8 caracteres","Se requiere la contraseña SSH"]
N["am"] = ["ዶሜይን (ወደዚህ ሰርቨር IP መጠቆም አለበት)","የFirefox መግቢያ ተጠቃሚ ስም","የFirefox መግቢያ የይለፍ ቃል (ቢያንስ 8)","የስክሪን ጥራት",
"ከመጫንህ በፊት የዶሜይኑን A ሪከርድ ወደ ሰርቨሩ IP ጠቁምና ፖርት 80 እና 443ን ክፈት። ስርዓት: Ubuntu 22.04 / 24.04 ወይም Debian 12።",
"Firefox ክፈት ↗","Firefox ጫን","Firefox እንደገና አስጀምር","Firefox ከ{name} ይወገድ?",
"{name}\n1 = Firefoxን ከሰርቨሩም አስወግድ፣ ከዚያ ከዝርዝሩ ሰርዝ\n2 = ከዝርዝሩ ብቻ ሰርዝ",
"የሰርቨሩ IP ወይም አድራሻ ልክ አይደለም","ዶሜይኑ ልክ አይደለም (ምሳሌ: fire.example.com)",
"የFirefox ተጠቃሚ ስም: ከ3 እስከ 32 ፊደላት፣ ቁጥሮች፣ ነጥብ፣ ሰረዝ ወይም ከስር ሰረዝ","የFirefox የይለፍ ቃል ቢያንስ 8 ቁምፊዎች መሆን አለበት","SSH የይለፍ ቃል ያስፈልጋል"]
N["ti"] = ["ዶሜይን (ናብ IP እዚ ሰርቨር ክሐብር ኣለዎ)","ስም ተጠቃሚ ምእታው Firefox","መሕለፊ ቃል ምእታው Firefox (እንተወሓደ 8)","ጽሬት ስክሪን",
"ቅድሚ ምእታው፡ A ሪከርድ ናይቲ ዶሜይን ናብ IP ሰርቨር ኣመልክት፡ ፖርት 80ን 443ን ክፈት። ስርዓት: Ubuntu 22.04 / 24.04 ወይ Debian 12።",
"Firefox ክፈት ↗","Firefox ኣእቱ","Firefox ዳግማይ ጀምር","Firefox ካብ {name} ይእለ ዶ?",
"{name}\n1 = Firefox ካብ ሰርቨር እውን ኣልዮ፣ ድሕሪኡ ካብ ዝርዝር ሰርዝ\n2 = ካብ ዝርዝር ጥራይ ሰርዝ",
"IP ወይ ኣድራሻ ሰርቨር ቅኑዕ ኣይኮነን","ዶሜይን ቅኑዕ ኣይኮነን (ኣብነት: fire.example.com)",
"ስም ተጠቃሚ Firefox: ካብ 3 ክሳብ 32 ፊደላት፡ ቁጽርታት፡ ነጥቢ፡ ሰረዝ ወይ ታሕተዋይ ሰረዝ","መሕለፊ ቃል Firefox እንተወሓደ 8 ፊደላት ክኸውን ኣለዎ","SSH መሕለፊ ቃል የድሊ"]

KEYS = REUSE + NEW
T = {}
for c, *_ in LANGS:
    b = dict(zip(BK, BT[c]))
    assert len(N[c]) == len(NEW), c
    T[c] = [b[k] for k in REUSE] + N[c]
