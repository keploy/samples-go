package utils

import (
	"strings"

	"github.com/google/uuid"
)

func GeneratorUUID(strLen int) string {
	return strings.Replace(uuid.NewString(), "-", "", -1)[:strLen]
}
