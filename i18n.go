package main

// Lang is a supported bot language.
type Lang string

const (
	LangEN Lang = "en"
	LangAM Lang = "am"
)

// userLang maps userID -> the language they picked at /start.
// Users who haven't picked one yet (or after a restart) fall back to LangEN.
var userLang = make(map[int64]Lang)

func getLang(userID int64) Lang {
	if lang, ok := userLang[userID]; ok {
		return lang
	}
	return LangEN
}

// translations holds every user-facing string keyed by a short identifier,
// with one variant per supported language.
var translations = map[string]map[Lang]string{
	"btn_lang_en": {
		LangEN: "🇬🇧 English",
		LangAM: "🇬🇧 English",
	},
	"btn_lang_am": {
		LangEN: "🇪🇹 አማርኛ",
		LangAM: "🇪🇹 አማርኛ",
	},
	"language_prompt": {
		LangEN: "🇪🇹 Welcome to Temari!\nPlease choose your language below.\n\nእንኳን ወደ ተማሪ በደህና መጡ!\nእባክዎ ቋንቋዎን ከታች ይምረጡ።",
		LangAM: "🇪🇹 Welcome to Temari!\nPlease choose your language below.\n\nእንኳን ወደ ተማሪ በደህና መጡ!\nእባክዎ ቋንቋዎን ከታች ይምረጡ።",
	},
	"contact_request": {
		LangEN: "To proceed, please send us a *screenshot of your profile page ON THE TEMARI APP* (showing your registered Email or Phone Number).\n\n" +
			"OR, \n\n you can just type the *Email* or *Phone Number* you used to register on the Temari App here directly.\n\n" +
			"Example: `0912345678` or `example@email.com`",
		LangAM: "ለመቀጠል፣ እባክዎ *በTEMARI APP* ላይ *የPROFILE PAGE SCREENSHOT* ይላኩ።\n\n" +
			"ወይም፣ \n\n በተማሪ መተግበሪያ ላይ የተመዘገቡበትን *EMAIL ወይም PHONE NUMBER* በቀጥታ እዚህ መጻፍ ይችላሉ።\n\n" +
			"ምሳሌ፦ `0912345678` ወይም `example@email.com`",
	},
	"payment_instructions": {
		LangEN: "To get Premium access:\n\n" +
			"1. Transfer 150 ETB to *one* of the following accounts:\n\n" +
			"🏦 *CBE*\n" +
			"`1000415850388`\n" +
			"_Name_: Natanim Ashenafi\n\n" +
			"📱 *Telebirr*\n" +
			"`0983082255`\n" +
			"_Name_: Natanim Ashenafi\n\n" +
			"2. *Send a SCREENSHOT of your receipt here.*",
		LangAM: "Premium Access ለማግኘት፦\n\n" +
			"1. 150 ብር ከሚከተሉት አካውንቶች ወደ *አንዱ* ያስተላልፉ፦\n\n" +
			"🏦 *ንግድ ባንክ (CBE)*\n" +
			"`1000415850388`\n" +
			"_ስም_: Natanim Ashenafi\n\n" +
			"📱 *ቴሌብር*\n" +
			"`0983082255`\n" +
			"_ስም_: Natanim Ashenafi\n\n" +
			"2. *የክፍያ ደረሰኝዎን ስክሪንሾት እዚህ ይላኩ።*",
	},
	"profile_screenshot_received": {
		LangEN: "✅ Got your profile screenshot!",
		LangAM: "✅ የፕሮፋይል ስክሪንሾትዎ ደርሶናል!",
	},
	"payment_screenshot_received": {
		LangEN: "✅ Got your payment screenshot! Now, let's link it to your account.",
		LangAM: "✅ የክፍያ ስክሪንሾትዎ ደርሶናል! አሁን ከአካውንትዎ ጋር እናገናኘው።",
	},
	"profile_screenshot_error": {
		LangEN: "Something went wrong while receiving your screenshot. Please try again or contact @TemariAppSupport.",
		LangAM: "ስክሪንሾትዎን በመቀበል ላይ ችግር ተፈጥሯል። እባክዎ እንደገና ይሞክሩ ወይም @TemariAppSupport ያግኙ።",
	},
	"proof_sent": {
		LangEN: "✅ *Proof Sent!*\n\nOur team is now verifying your payment. You will receive a notification here once your account is activated.",
		LangAM: "✅ *ደረሰኝ ተልኳል!*\n\nቡድናችን ክፍያዎን በማረጋገጥ ላይ ነው። አካውንትዎ ሲነቃ እዚሁ ማሳወቂያ ይደርስዎታል።",
	},
	"proof_error": {
		LangEN: "Something went wrong while sending your proof. Please contact @TemariAppSupport.",
		LangAM: "ደረሰኝዎን በመላክ ላይ ችግር ተፈጥሯል። እባክዎ @TemariAppSupport ያግኙ።",
	},
	"doc_not_photo": {
		LangEN: "Please send the receipt as a **Photo** (Image), not as a file/document. This helps us verify it faster!",
		LangAM: "እባክዎ ደረሰኙን እንደ **ፎቶ** (ምስል) ይላኩ፤ እንደ ፋይል/ሰነድ አይላኩ። ይህ በፍጥነት እንድናረጋግጥ ይረዳናል!",
	},
	"invalid_contact": {
		LangEN: "❌ I didn't recognize that as a valid Email or Ethiopian Phone Number. Please try again.",
		LangAM: "❌ ይህ ትክክለኛ ኢሜይል ወይም የኢትዮጵያ ስልክ ቁጥር ሆኖ አላገኘሁትም። እባክዎ እንደገና ይሞክሩ።",
	},
	"linked_contact": {
		LangEN: "✅ Linked to %s: `%s`",
		LangAM: "✅ ከ%s ጋር ተያይዟል፦ `%s`",
	},
	"contact_type_email": {
		LangEN: "email",
		LangAM: "ኢሜይል",
	},
	"contact_type_phone": {
		LangEN: "phone number",
		LangAM: "ስልክ ቁጥር",
	},
	"access_granted": {
		LangEN: "🎉 *Access Granted!*\n\n" +
			"Your premium access has been activated. Please **close the app and open it again** to gain full access.\n\n" +
			"Join our channel for latest updates: [Temari Channel](https://t.me/temariapp)\n\n" +
			"If you have any issues, contact @TemariAppSupport.",
		LangAM: "🎉 *Access Granted!*\n\n" +
			"የፕሪሚየም መዳረሻዎ ነቅቷል። ሙሉ መዳረሻ ለማግኘት እባክዎ **መተግበሪያውን ዘግተው እንደገና ይክፈቱት**።\n\n" +
			"ለቅርብ ጊዜ ዝማኔዎች ቻናላችንን ይቀላቀሉ፦ [Temari Channel](https://t.me/temariapp)\n\n" +
			"ማንኛውም ችግር ካጋጠመዎት፣ @TemariAppSupport ያግኙ።",
	},
}

// t returns the translation for key in lang, falling back to English if the
// language variant is missing, and to the key itself if the key is unknown.
func t(lang Lang, key string) string {
	variants, ok := translations[key]
	if !ok {
		return key
	}
	if s, ok := variants[lang]; ok {
		return s
	}
	return variants[LangEN]
}
