package yamlintegration

import "time"

// DateVariables devuelve variables de plantilla con fechas relativas a now.
// Estas variables se mezclan automáticamente en cada ejecución de Run.
//
//   TODAY                  - fecha actual en formato YYYY-MM-DD
//   FIRST_DAY_CURRENT_MONTH
//   FIRST_DAY_PREVIOUS_MONTH
//   LAST_DAY_PREVIOUS_MONTH
//   TWO_DAYS_FROM_NOW
//   FUTURE_DATE            - fecha lejana (3000-01-01)
//   PAST_DATE              - fecha pasada (2000-01-01)
func DateVariables(now time.Time) map[string]string {
	firstDayCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstDayPreviousMonth := firstDayCurrentMonth.AddDate(0, -1, 0)
	lastDayPreviousMonth := firstDayCurrentMonth.AddDate(0, 0, -1)
	twoDaysFromNow := now.AddDate(0, 0, 2)

	return map[string]string{
		"TODAY":                   now.Format("2006-01-02"),
		"FIRST_DAY_CURRENT_MONTH": firstDayCurrentMonth.Format("2006-01-02"),
		"FIRST_DAY_PREVIOUS_MONTH": firstDayPreviousMonth.Format("2006-01-02"),
		"LAST_DAY_PREVIOUS_MONTH": lastDayPreviousMonth.Format("2006-01-02"),
		"TWO_DAYS_FROM_NOW":       twoDaysFromNow.Format("2006-01-02"),
		"FUTURE_DATE":             "3000-01-01",
		"PAST_DATE":               "2000-01-01",
	}
}
