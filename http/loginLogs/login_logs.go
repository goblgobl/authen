package loginLogs

import "src.goblgobl.com/utils/validation"

var (
	userIdValidation = validation.String().Required().Length(1, 100)
	statusValidation = validation.Int()
)
