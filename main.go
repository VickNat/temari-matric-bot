package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

// Buttons are declared here only to give Handle(&btn, ...) a stable Unique
// to match against; their Text is set per-language when the keyboard is built.
var (
	btnLangEN     = tele.Btn{Unique: "lang_en"}
	btnLangAM     = tele.Btn{Unique: "lang_am"}
	btnConfirmPay = tele.Btn{Unique: "grant_access"}
)

func buildLanguageMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	enBtn := m.Data(t(LangEN, "btn_lang_en"), btnLangEN.Unique)
	amBtn := m.Data(t(LangEN, "btn_lang_am"), btnLangAM.Unique)
	m.Inline(m.Row(enBtn, amBtn))
	return m
}

// A map to store user data userid -> phone number or email
var userStore = make(map[int64]string)

// pendingProfileScreenshot holds a user's profile screenshot (received at the
// contact-request step) until their payment proof screenshot arrives, so the
// two can be forwarded to the admin together as one album.
var pendingProfileScreenshot = make(map[int64]tele.File)

// adminID is set in main(), after .env is loaded — a package-level initializer
// would run before godotenv.Load() and always see an empty ADMIN_ID.
var adminID int64

func validateContact(input string) (string, bool) {
	input = strings.TrimSpace(input)

	// 1. Email Regex (Standard)
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

	// 2. Ethiopian Phone Regex
	// Matches: 0912345678, 0712345678, +251912345678, 251912345678
	phoneRegex := regexp.MustCompile(`^(?:\+251|251|0)?([79]\d{8})$`)

	if emailRegex.MatchString(strings.ToLower(input)) {
		return "email", true
	}

	if phoneRegex.MatchString(input) {
		return "phone", true
	}

	return "", false
}

func generateTelegramUserLink(userName string, userID int64) string {
	if userName != "" {
		return "@" + userName
	}
	return fmt.Sprintf(`No username available. User ID: %d`, userID)
}

func sendLanguageSelection(c tele.Context) error {
	return c.Send(t(LangEN, "language_prompt"), buildLanguageMenu())
}

func sendContactRequest(c tele.Context, lang Lang) error {
	contactRequest := t(lang, "contact_request")

	if c.Callback() != nil {
		return c.Edit(contactRequest, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	}

	return c.Send(contactRequest, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

func sendPaymentInstructions(c tele.Context, lang Lang) error {
	paymentInstructions := t(lang, "payment_instructions")

	if c.Callback() != nil {
		return c.Edit(paymentInstructions, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	}

	return c.Send(paymentInstructions, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

// handleLanguageChosen records the user's language pick, then continues straight
// into payment instructions (if we already have their contact info, e.g. from a
// valid /start payload) or the contact request, in that language.
func handleLanguageChosen(c tele.Context, lang Lang) error {
	userID := c.Sender().ID
	userLang[userID] = lang

	if _, exists := userStore[userID]; exists {
		return sendPaymentInstructions(c, lang)
	}

	return sendContactRequest(c, lang)
}

func contactTypeLabel(lang Lang, contactType string) string {
	switch contactType {
	case "email":
		return t(lang, "contact_type_email")
	case "phone":
		return t(lang, "contact_type_phone")
	default:
		return contactType
	}
}

func main() {
	// Silently fail if the .env file is not found
	_ = godotenv.Load()

	parsedAdminID, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	if err != nil {
		log.Fatal("ADMIN_ID env var is missing or invalid: ", err)
	}
	adminID = parsedAdminID

	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 1. /start always opens with the language picker. A valid contact from the
	// payload is stored right away so it's ready the moment a language is picked.
	b.Handle("/start", func(ctx tele.Context) error {
		payload := ctx.Message().Payload
		userID := ctx.Message().Sender.ID

		if payload != "" {
			decodedPayload, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				log.Println("Error decoding payload:", err)
			} else {
				decoded := string(decodedPayload)
				log.Println("Decoded payload:", decoded)

				if _, isValid := validateContact(decoded); isValid {
					userStore[userID] = decoded
					log.Println("User", userID, "has provided email or phone number:", decoded)
				} else {
					log.Println("User", userID, "has provided INVALID email or phone number:", decoded)
				}
			}
		}

		return sendLanguageSelection(ctx)
	})

	// 2. Handle language selection
	b.Handle(&btnLangEN, func(c tele.Context) error {
		return handleLanguageChosen(c, LangEN)
	})
	b.Handle(&btnLangAM, func(c tele.Context) error {
		return handleLanguageChosen(c, LangAM)
	})

	// 3. Handle the screenshot upload
	b.Handle(tele.OnPhoto, func(c tele.Context) error {
		userID := c.Message().Sender.ID
		userName := c.Message().Sender.Username
		lang := getLang(userID)

		contactInfo, exists := userStore[userID]

		// If we don't have their email or phone number yet, treat this photo as their
		// Temari app profile screenshot. We hold onto it rather than forwarding it
		// right away, so it can be sent to the admin together with the payment proof
		// once that arrives.
		if !exists {
			pendingProfileScreenshot[userID] = c.Message().Photo.File
			userStore[userID] = "[Profile Screenshot]"

			if err := c.Send(t(lang, "profile_screenshot_received")); err != nil {
				return err
			}
			return sendPaymentInstructions(c, lang)
		}

		// If we have their email or phone number, we can proceed with the payment verification
		caption := fmt.Sprintf("🚨 <b>NEW PAYMENT PROOF</b>\n\n"+
			"<b>User:</b> %s\n"+
			"<b>ID:</b> <code>%d</code>\n"+
			"<b>Account:</b> <code>%s</code>",
			generateTelegramUserLink(userName, userID), userID, contactInfo)

		adminMarkup := &tele.ReplyMarkup{}
		btnWithData := btnConfirmPay
		btnWithData.Data = fmt.Sprintf("%d", userID)

		adminMarkup.Inline(adminMarkup.Row(btnWithData))

		paymentPhoto := &tele.Photo{File: c.Message().Photo.File}

		if profileFile, hasProfileScreenshot := pendingProfileScreenshot[userID]; hasProfileScreenshot {
			// Send both screenshots together as one album, caption on the first item.
			profilePhoto := &tele.Photo{File: profileFile, Caption: caption}

			if _, err := b.SendAlbum(tele.ChatID(adminID), tele.Album{profilePhoto, paymentPhoto}, &tele.SendOptions{ParseMode: tele.ModeHTML}); err != nil {
				log.Println("Failed to forward payment proof album to admin:", err)
				return c.Send(t(lang, "proof_error"))
			}
			delete(pendingProfileScreenshot, userID)

			// Media groups can't carry an inline keyboard, so the Confirm button
			// goes out as its own follow-up message.
			if _, err := b.Send(tele.ChatID(adminID), "⬆️ Tap below to confirm the payment above.", adminMarkup); err != nil {
				log.Println("Failed to send confirm-payment button to admin:", err)
			}
		} else {
			paymentPhoto.Caption = caption

			if _, err := b.Send(tele.ChatID(adminID), paymentPhoto, &tele.SendOptions{
				ParseMode:   tele.ModeHTML,
				ReplyMarkup: adminMarkup,
			}); err != nil {
				log.Println("Failed to forward to admin:", err)
				return c.Send(t(lang, "proof_error"))
			}
		}

		return c.Send(t(lang, "proof_sent"), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	})

	// 4. Handle non-photo messages (Error handling)
	b.Handle(tele.OnDocument, func(c tele.Context) error {
		return c.Send(t(getLang(c.Sender().ID), "doc_not_photo"))
	})

	// 5. Handle manual text input for email/phone
	b.Handle(tele.OnText, func(c tele.Context) error {
		// Ignore commands like /start
		if strings.HasPrefix(c.Text(), "/") {
			return nil
		}

		userID := c.Sender().ID
		lang := getLang(userID)
		input := c.Text()

		contactType, isValid := validateContact(input)

		if isValid {
			userStore[userID] = input
			linkedMsg := fmt.Sprintf(t(lang, "linked_contact"), contactTypeLabel(lang, contactType), input)
			if err := c.Send(linkedMsg, &tele.SendOptions{ParseMode: tele.ModeMarkdown}); err != nil {
				return err
			}
			return sendPaymentInstructions(c, lang)
		}

		return c.Send(t(lang, "invalid_contact"))
	})

	// 6. Handle Admin clicking "Confirm Payment"
	b.Handle(&btnConfirmPay, func(c tele.Context) error {
		// 1. Extract the target userID from the callback data
		targetUserIDStr := c.Data()
		var targetUserID int64
		_, err := fmt.Sscanf(targetUserIDStr, "%d", &targetUserID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "❌ Error: Could not parse User ID"})
		}

		// 2. Notify the Student, in their chosen language
		successMsg := t(getLang(targetUserID), "access_granted")

		_, err = b.Send(tele.ChatID(targetUserID), successMsg, tele.ModeMarkdown)
		if err != nil {
			log.Println("Failed to notify user:", err)
			return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to message the student."})
		}

		// 2. Clear the user from the store so they can pay again/differently later
		// This is the line you need to add:
		delete(userStore, targetUserID)

		// The Confirm button sits on a photo caption (single-screenshot case) or a
		// plain text message (album case, since media groups can't carry buttons).
		statusSuffix := "\n\n✅ <b>STATUS: Access Granted</b>"
		if c.Message().Photo != nil {
			_, err = b.EditCaption(c.Message(), c.Message().Caption+statusSuffix, &tele.SendOptions{
				ParseMode: tele.ModeHTML,
			})
		} else {
			_, err = b.Edit(c.Message(), c.Message().Text+statusSuffix, &tele.SendOptions{
				ParseMode: tele.ModeHTML,
			})
		}

		if err != nil {
			log.Println("Failed to update admin message status:", err)
			// If editing fails, we still want to stop the loading spinner
		}

		// 3. Stop the loading spinner
		return c.Respond(&tele.CallbackResponse{Text: "Student notified!"})
	})

	log.Println("Temari is running")
	b.Start()
}
