package storage

func (bl *badgerLogger) Errorf(fmt string, args ...interface{}) {
	bl.Logger.Error().Msgf(fmt, args...)
}
func (bl *badgerLogger) Warningf(fmt string, args ...interface{}) {
	bl.Logger.Warn().Msgf(fmt, args...)
}
func (bl *badgerLogger) Infof(fmt string, args ...interface{}) {
	bl.Logger.Info().Msgf(fmt, args...)
}
func (bl *badgerLogger) Debugf(fmt string, args ...interface{}) {
	bl.Logger.Debug().Msgf(fmt, args)
}
