package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

var (
	// Main Menu Keyboard
	menu       = &tele.ReplyMarkup{}
	btnPay     = menu.Data("💳 Pay 150 ETB", "pay_flow")
	btnSupport = menu.URL("📞 Contact Support", "https://t.me/TemariAppSupport")
)

var (
	// Admin Approval Keyboard
	adminMenu     = &tele.ReplyMarkup{}
	btnConfirmPay = adminMenu.Data("✅ Confirm Payment", "grant_access")
)

// A map to store user data userid -> phone number or email
var userStore = make(map[int64]string)

var adminID = os.Getenv("ADMIN_ID")

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

func sendPaymentInstructions(c tele.Context) error {
	paymentInstructions := "To get Premium access:\n\n" +
		"1. Transfer 150 ETB to *one* of the following accounts:\n\n" +
		"🏦 *CBE*\n" +
		"`1000415850388`\n" +
		"_Name_: Natanim Ashenafi\n\n" +
		"📱 *Telebirr*\n" +
		"`0983082255`\n" +
		"_Name_: Natanim Ashenafi\n\n" +
		"2. *Send a SCREENSHOT of your receipt here.*"

	if c.Callback() != nil {
		return c.Edit(paymentInstructions, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	}

	return c.Send(paymentInstructions, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 1. Define the Main Menu layout
	menu.Inline(
		menu.Row(btnPay),
		menu.Row(btnSupport),
	)

	// 2. Update your /start handler to show the menu
	b.Handle("/start", func(ctx tele.Context) error {
		payload := ctx.Message().Payload
		userID := ctx.Message().Sender.ID

		if payload != "" {
			decodedPayload, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				log.Println("Error decoding payload:", err)
				return ctx.Send("Error processing your request. Please try again.", menu)
			}

			payload = string(decodedPayload)

			log.Println("Decoded payload:", payload)
			_, isValid := validateContact(payload)

			if isValid {
				userStore[userID] = payload
				log.Println("User", userID, "has provided email or phone number:", payload)
				return sendPaymentInstructions(ctx)
			}
			log.Println("User", userID, "has provided INVALID email or phone number:", payload)
			// If the payload is not a valid email or phone number, continue with the normal flow
		}

		return ctx.Send("Hello Temari! 🇪🇹\nUnlock all past Matric exams for 150 ETB.", menu)
	})

	// 3. Handle the "Pay" button click
	b.Handle(&btnPay, func(c tele.Context) error {
		return sendPaymentInstructions(c)
	})

	// 4. Handle the screenshot upload
	b.Handle(tele.OnPhoto, func(c tele.Context) error {
		userID := c.Message().Sender.ID
		userName := c.Message().Sender.Username

		contactInfo, exists := userStore[userID]

		// If we don't have their email or phone number yet, we accept the screenshot and ask them to send their contact info
		if !exists {
			return c.Send("I've received your screenshot, but I don't have your account info yet. Please type your Email or Phone Number first, then send the screenshot again!")
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

		photo := &tele.Photo{
			File:    c.Message().Photo.File,
			Caption: caption,
		}

		// Forward to admin
		_, err := b.Send(tele.ChatID(adminID), photo, &tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: adminMarkup,
		})

		if err != nil {
			log.Println("Failed to forward to admin:", err)
			return c.Send("Something went wrong while sending your proof. Please contact @TemariAppSupport.")
		}

		return c.Send("✅ *Proof Sent!*\n\nOur team is now verifying your payment. You will receive a notification here once your account is activated.")
	})

	// 5. Handle non-photo messages (Error handling)
	b.Handle(tele.OnDocument, func(c tele.Context) error {
		return c.Send("Please send the receipt as a **Photo** (Image), not as a file/document. This helps us verify it faster!")
	})

	// 6. Handle manual text input for email/phone
	b.Handle(tele.OnText, func(c tele.Context) error {
		// Ignore commands like /start
		if strings.HasPrefix(c.Text(), "/") {
			return nil
		}

		userID := c.Sender().ID
		input := c.Text()

		contactType, isValid := validateContact(input)

		if isValid {
			userStore[userID] = input
			return c.Send(fmt.Sprintf("✅ Linked to %s: `%s`\n\nNow, please upload the payment screenshot to finish.", contactType, input))
		}

		return c.Send("❌ I didn't recognize that as a valid Email or Ethiopian Phone Number. Please try again.")
	})

	// 7. Handle Admin clicking "Confirm Payment"
	b.Handle(&btnConfirmPay, func(c tele.Context) error {
		// 1. Extract the target userID from the callback data
		targetUserIDStr := c.Data()
		var targetUserID int64
		_, err := fmt.Sscanf(targetUserIDStr, "%d", &targetUserID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "❌ Error: Could not parse User ID"})
		}

		// 2. Notify the Student
		successMsg := "🎉 *Access Granted!*\n\n" +
			"Your premium access has been activated. Please **close the app and open it again** to gain full access.\n\n" +
			"Join our channel for latest updates: [Temari Channel](https://t.me/your_channel)\n\n" +
			"If you have any issues, contact @TemariAppSupport."

		_, err = b.Send(tele.ChatID(targetUserID), successMsg, tele.ModeMarkdown)
		if err != nil {
			log.Println("Failed to notify user:", err)
			return c.Respond(&tele.CallbackResponse{Text: "❌ Failed to message the student."})
		}

		// 2. Clear the user from the store so they can pay again/differently later
		// This is the line you need to add:
		delete(userStore, targetUserID)

		newCaption := c.Message().Caption + "\n\n✅ <b>STATUS: Access Granted</b>"

		_, err = b.EditCaption(c.Message(), newCaption, &tele.SendOptions{
			ParseMode: tele.ModeHTML,
		})

		if err != nil {
			log.Println("EditCaption error:", err)
			// If editing fails, we still want to stop the loading spinner
		}

		// 3. Stop the loading spinner
		return c.Respond(&tele.CallbackResponse{Text: "Student notified!"})
	})

	log.Println("Temari is running")
	b.Start()
}
