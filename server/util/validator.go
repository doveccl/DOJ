package util

import "regexp"

var nameReg = regexp.MustCompile(`^\w{3,16}$`)
var mailReg = regexp.MustCompile(`^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`)

func ValidateName(name string) bool {
	return nameReg.MatchString(name)
}

func ValidateMail(mail string) bool {
	return mailReg.MatchString(mail)
}
