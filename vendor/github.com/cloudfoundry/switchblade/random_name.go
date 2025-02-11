package switchblade

import (
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

func RandomName() (string, error) {
	id, err := shortid.Generate()
	if err != nil {
		return "", err
	}

	return strings.ToLower(fmt.Sprintf("switchblade-%s", id)), nil
}
