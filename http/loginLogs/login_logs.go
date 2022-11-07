package loginLogs

import "src.goblgobl.com/utils/validation"

var (
	userIdValidation = validation.String("user_id").Required().Length(1, 100)
)
