package bore

import (
	"github.com/rs/zerolog/log"
)

func logError(err error) {
	log.Error().Err(err).Caller(1).Send()
}
