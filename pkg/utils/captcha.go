package utils

import "github.com/mojocn/base64Captcha"

func GenerateCaptchaImage() (string, string, string, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
	c := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	id, b64s, answer, err := c.Generate()
	return id, b64s, answer, err
}
