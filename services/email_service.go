// services/email_service.go
package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"

	"github.com/capactiyvirus/stripe-backend/models"
)

type EmailService struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type EmailData struct {
	Order        *models.Order
	TrackingURL  string
	SupportEmail string
	CompanyName  string
	DownloadURLs map[string]string // productID -> downloadURL
}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	return &EmailService{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("FROM_EMAIL"),
		FromName:     os.Getenv("FROM_NAME"),
	}
}

// SendOrderConfirmation sends order confirmation email
func (e *EmailService) SendOrderConfirmation(order *models.Order) error {
	subject := fmt.Sprintf("Order Confirmation - %s", order.TrackingID)

	data := EmailData{
		Order:        order,
		TrackingURL:  fmt.Sprintf("https://yourdomain.com/track-order?id=%s", order.TrackingID),
		SupportEmail: "support@yourdomain.com",
		CompanyName:  "PlannerPalette",
	}

	htmlBody, err := e.renderTemplate("order_confirmation.html", data)
	if err != nil {
		return err
	}

	return e.sendEmail(order.CustomerInfo.Email, subject, htmlBody)
}

// SendPaymentConfirmation sends payment confirmation email
func (e *EmailService) SendPaymentConfirmation(order *models.Order) error {
	subject := fmt.Sprintf("Payment Confirmed - %s", order.TrackingID)

	data := EmailData{
		Order:        order,
		TrackingURL:  fmt.Sprintf("https://yourdomain.com/track-order?id=%s", order.TrackingID),
		SupportEmail: "support@yourdomain.com",
		CompanyName:  "PlannerPalette",
	}

	htmlBody, err := e.renderTemplate("payment_confirmation.html", data)
	if err != nil {
		return err
	}

	return e.sendEmail(order.CustomerInfo.Email, subject, htmlBody)
}

// SendFulfillmentEmail sends order fulfillment email with download links
func (e *EmailService) SendFulfillmentEmail(order *models.Order, downloadURLs map[string]string) error {
	subject := fmt.Sprintf("Your Order is Ready for Download - %s", order.TrackingID)

	data := EmailData{
		Order:        order,
		TrackingURL:  fmt.Sprintf("https://yourdomain.com/track-order?id=%s", order.TrackingID),
		SupportEmail: "support@yourdomain.com",
		CompanyName:  "PlannerPalette",
		DownloadURLs: downloadURLs,
	}

	htmlBody, err := e.renderTemplate("order_fulfillment.html", data)
	if err != nil {
		return err
	}

	return e.sendEmail(order.CustomerInfo.Email, subject, htmlBody)
}

// SendRefundNotification sends refund notification email
func (e *EmailService) SendRefundNotification(order *models.Order) error {
	subject := fmt.Sprintf("Refund Processed - %s", order.TrackingID)

	data := EmailData{
		Order:        order,
		TrackingURL:  fmt.Sprintf("https://yourdomain.com/track-order?id=%s", order.TrackingID),
		SupportEmail: "support@yourdomain.com",
		CompanyName:  "PlannerPalette",
	}

	htmlBody, err := e.renderTemplate("refund_notification.html", data)
	if err != nil {
		return err
	}

	return e.sendEmail(order.CustomerInfo.Email, subject, htmlBody)
}

// renderTemplate renders an email template with data
func (e *EmailService) renderTemplate(templateName string, data EmailData) (string, error) {
	// Get template content based on template name
	templateContent := e.getEmailTemplate(templateName)

	tmpl, err := template.New(templateName).Parse(templateContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// sendEmail sends an email using SMTP
func (e *EmailService) sendEmail(to, subject, htmlBody string) error {
	// Create the email message
	msg := e.buildEmailMessage(to, subject, htmlBody)

	// Connect to SMTP server
	auth := smtp.PlainAuth("", e.SMTPUsername, e.SMTPPassword, e.SMTPHost)

	// Send the email
	err := smtp.SendMail(
		e.SMTPHost+":"+e.SMTPPort,
		auth,
		e.FromEmail,
		[]string{to},
		[]byte(msg),
	)

	return err
}

// buildEmailMessage builds the email message with headers
func (e *EmailService) buildEmailMessage(to, subject, htmlBody string) string {
	from := fmt.Sprintf("%s <%s>", e.FromName, e.FromEmail)

	msg := fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += htmlBody

	return msg
}

// getEmailTemplate returns email template content
func (e *EmailService) getEmailTemplate(templateName string) string {
	switch templateName {
	case "order_confirmation.html":
		return orderConfirmationTemplate
	case "payment_confirmation.html":
		return paymentConfirmationTemplate
	case "order_fulfillment.html":
		return orderFulfillmentTemplate
	case "refund_notification.html":
		return refundNotificationTemplate
	default:
		return basicEmailTemplate
	}
}

// Email Templates
const orderConfirmationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Order Confirmation</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; }
        .header { background: #2c3b3a; color: white; padding: 20px; text-align: center; }
        .content { background: white; padding: 30px; }
        .order-details { background: #f5f5f5; padding: 20px; margin: 20px 0; }
        .item { border-bottom: 1px solid #eee; padding: 10px 0; }
        .total { font-weight: bold; font-size: 18px; margin-top: 10px; }
        .tracking { background: #e8f4f8; padding: 15px; margin: 20px 0; border-left: 5px solid #2c3b3a; }
        .button { background: #6e725a; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.CompanyName}}</h1>
            <h2>Order Confirmation</h2>
        </div>
        
        <div class="content">
            <p>Hi {{.Order.CustomerInfo.Name}},</p>
            
            <p>Thank you for your order! We've received your purchase and are processing it now.</p>
            
            <div class="tracking">
                <strong>Your Tracking ID: {{.Order.TrackingID}}</strong><br>
                Use this ID to track your order status at any time.
            </div>
            
            <div class="order-details">
                <h3>Order Details</h3>
                <p><strong>Order ID:</strong> {{.Order.ID}}</p>
                <p><strong>Date:</strong> {{.Order.CreatedAt.Format "January 2, 2006"}}</p>
                
                <h4>Items Ordered:</h4>
                {{range .Order.Items}}
                <div class="item">
                    <strong>{{.ProductName}}</strong><br>
                    {{.FileType}} • Quantity: {{.Quantity}}<br>
                    Price: ${{printf "%.2f" .Price}}
                </div>
                {{end}}
                
                <div class="total">
                    Total: ${{printf "%.2f" (div .Order.Payment.Amount 100.0)}}
                </div>
            </div>
            
            <p>You will receive another email once your payment is confirmed and your order is ready for download.</p>
            
            <a href="{{.TrackingURL}}" class="button">Track Your Order</a>
            
            <p>If you have any questions, please contact us at {{.SupportEmail}}.</p>
            
            <p>Thank you for choosing {{.CompanyName}}!</p>
        </div>
        
        <div class="footer">
            <p>&copy; {{.CompanyName}} - Empowering Writers Worldwide</p>
        </div>
    </div>
</body>
</html>
`

const paymentConfirmationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Payment Confirmed</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; }
        .header { background: #2c3b3a; color: white; padding: 20px; text-align: center; }
        .content { background: white; padding: 30px; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .tracking { background: #e8f4f8; padding: 15px; margin: 20px 0; border-left: 5px solid #2c3b3a; }
        .button { background: #6e725a; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.CompanyName}}</h1>
            <h2>Payment Confirmed! 🎉</h2>
        </div>
        
        <div class="content">
            <div class="success">
                <strong>Great news!</strong> Your payment has been successfully processed.
            </div>
            
            <p>Hi {{.Order.CustomerInfo.Name}},</p>
            
            <p>Your payment of <strong>${{printf "%.2f" (div .Order.Payment.Amount 100.0)}}</strong> has been confirmed for order {{.Order.TrackingID}}.</p>
            
            <div class="tracking">
                <strong>What's Next?</strong><br>
                We're now preparing your digital downloads. You'll receive an email with download links within the next few hours.
            </div>
            
            <a href="{{.TrackingURL}}" class="button">Track Your Order</a>
            
            <p>Thank you for your business!</p>
        </div>
        
        <div class="footer">
            <p>&copy; {{.CompanyName}} - Crafting Stories, One Guide at a Time</p>
        </div>
    </div>
</body>
</html>
`

const orderFulfillmentTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Downloads Are Ready!</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; }
        .header { background: #2c3b3a; color: white; padding: 20px; text-align: center; }
        .content { background: white; padding: 30px; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 20px; border-radius: 5px; margin: 20px 0; text-align: center; }
        .downloads { background: #f8f9fa; padding: 20px; margin: 20px 0; border-radius: 5px; }
        .download-item { background: white; padding: 15px; margin: 10px 0; border-radius: 5px; border: 1px solid #ddd; }
        .download-button { background: #6e725a; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block; }
        .important { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.CompanyName}}</h1>
            <h2>Your Downloads Are Ready! 📥</h2>
        </div>
        
        <div class="content">
            <div class="success">
                <h3>🎉 Order Complete!</h3>
                <p>Your writing resources are ready for download.</p>
            </div>
            
            <p>Hi {{.Order.CustomerInfo.Name}},</p>
            
            <p>Great news! Your order <strong>{{.Order.TrackingID}}</strong> has been processed and your digital writing guides are ready for download.</p>
            
            <div class="downloads">
                <h3>Your Downloads:</h3>
                {{range .Order.Items}}
                <div class="download-item">
                    <strong>{{.ProductName}}</strong><br>
                    Format: {{.FileType}}<br>
                    {{if index $.DownloadURLs .ProductID}}
                    <a href="{{index $.DownloadURLs .ProductID}}" class="download-button">Download {{.FileType}}</a>
                    {{else}}
                    <span style="color: #666;">Download link will be available shortly</span>
                    {{end}}
                </div>
                {{end}}
            </div>
            
            <div class="important">
                <strong>Important:</strong> Download links are valid for 30 days. Please save your files to your device. If you need to re-download after this period, please contact us.
            </div>
            
            <p>We hope these resources help you craft amazing stories! If you have any questions or need support, please don't hesitate to contact us at {{.SupportEmail}}.</p>
            
            <p>Happy writing!</p>
        </div>
        
        <div class="footer">
            <p>&copy; {{.CompanyName}} - Empowering Your Creative Journey</p>
        </div>
    </div>
</body>
</html>
`

const refundNotificationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Refund Processed</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; }
        .header { background: #2c3b3a; color: white; padding: 20px; text-align: center; }
        .content { background: white; padding: 30px; }
        .refund-info { background: #e3f2fd; border: 1px solid #bbdefb; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.CompanyName}}</h1>
            <h2>Refund Processed</h2>
        </div>
        
        <div class="content">
            <p>Hi {{.Order.CustomerInfo.Name}},</p>
            
            <p>We've processed a refund for your order <strong>{{.Order.TrackingID}}</strong>.</p>
            
            <div class="refund-info">
                <h3>Refund Details:</h3>
                <p><strong>Order ID:</strong> {{.Order.TrackingID}}</p>
                <p><strong>Refund Amount:</strong> ${{printf "%.2f" (div .Order.Payment.Amount 100.0)}}</p>
                <p><strong>Original Payment Method:</strong> Card ending in ****</p>
                <p><strong>Processing Time:</strong> 3-5 business days</p>
            </div>
            
            <p>The refund will appear on your original payment method within 3-5 business days, depending on your bank or card issuer.</p>
            
            <p>If you have any questions about this refund, please contact us at {{.SupportEmail}}.</p>
            
            <p>Thank you for your understanding.</p>
        </div>
        
        <div class="footer">
            <p>&copy; {{.CompanyName}} - Customer Service</p>
        </div>
    </div>
</body>
</html>
`

const basicEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.CompanyName}}</title>
</head>
<body>
    <h2>{{.CompanyName}}</h2>
    <p>This is a basic email template.</p>
</body>
</html>
`
