package formatter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/asmisnik/flats-analyzer/internal/model"
	"github.com/asmisnik/flats-analyzer/internal/region"
)

// FormatFlat renders a notification message for a flat. showRegion should be
// true when the recipient has active subscriptions in more than one region,
// and showDealType should be true when the recipient has active
// subscriptions of more than one deal type (rent and sale) — in both cases
// the message needs to clarify which region/deal type this flat belongs to.
func FormatFlat(f *model.FlatInfo, showRegion, showDealType bool) string {
	var sb strings.Builder

	sb.WriteString("🏠 Новая квартира")
	if showDealType {
		if f.DealType == "sale" {
			sb.WriteString(" на продажу")
		} else {
			sb.WriteString(" в аренду")
		}
	}
	sb.WriteString("\n\n")
	if showRegion {
		if name := region.Name(f.Region); name != "" {
			sb.WriteString("Регион: " + name + "\n")
		}
	}
	sb.WriteString("Ссылка: " + f.Link + "\n")
	if f.DealType == "sale" {
		fmt.Fprintf(&sb, "Цена: %s ₽\n", formatThousands(f.Price))
	} else {
		fmt.Fprintf(&sb, "Цена: %s ₽/мес\n", formatThousands(f.Price))
	}

	if f.Deposit > 0 {
		fmt.Fprintf(&sb, "Депозит: %s ₽", formatThousands(f.Deposit))
		if f.DepositMonths > 0 {
			fmt.Fprintf(&sb, " (%d мес. предоплаты)", f.DepositMonths)
		}
		sb.WriteString("\n")
	}
	if f.Comission > 0 {
		fmt.Fprintf(&sb, "Комиссия: %d%%\n", f.Comission)
	}

	sb.WriteString("\n")
	fmt.Fprintf(&sb, "Комнат: %d\n", f.RoomNumber)
	fmt.Fprintf(&sb, "Площадь: %.0f м² (жилая %.0f, кухня %.0f)\n",
		f.TotalArea, f.LivingArea, f.KitchenArea)
	fmt.Fprintf(&sb, "Этаж: %d из %d\n", f.Floor, f.MaxFloor)
	if f.CeilingHeight > 0 {
		fmt.Fprintf(&sb, "Высота потолков: %.1f м\n", f.CeilingHeight)
	}

	sb.WriteString("\n")
	if f.UndergroundDistanceInfo != "" {
		sb.WriteString("Метро: " + strings.TrimRight(f.UndergroundDistanceInfo, ", ") + "\n")
		fmt.Fprintf(&sb, "Место метро в рейтинге: %d\n", f.UndergroundPlace)
	}

	sb.WriteString("\n")
	if f.Renovation != "" {
		sb.WriteString("Ремонт: " + f.Renovation + "\n")
	}

	extras := []string{}
	if f.HasConditioner {
		extras = append(extras, "кондиционер")
	}
	if f.HasDishwasher {
		extras = append(extras, "посудомойка")
	}
	if f.LoggiaCount > 0 {
		extras = append(extras, fmt.Sprintf("лоджий: %d", f.LoggiaCount))
	}
	if f.BalconyCount > 0 {
		extras = append(extras, fmt.Sprintf("балконов: %d", f.BalconyCount))
	}
	if len(extras) > 0 {
		sb.WriteString("Удобства: " + strings.Join(extras, ", ") + "\n")
	}

	allowed := []string{}
	if f.ChildrenAllowed {
		allowed = append(allowed, "дети")
	}
	if f.PetsAllowed {
		allowed = append(allowed, "животные")
	}
	if len(allowed) > 0 {
		sb.WriteString("Разрешено: " + strings.Join(allowed, ", ") + "\n")
	}

	sb.WriteString("\n")
	fmt.Fprintf(&sb, "Score: %s\n", formatThousands(f.FlatScore))
	if f.LastUpdated != "" {
		sb.WriteString("Обновлено: " + f.LastUpdated + "\n")
	}

	return sb.String()
}

// formatThousands renders an integer with apostrophes between every group of
// three digits (e.g. 20000000 -> "20'000'000").
func formatThousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	var groups []string
	for len(s) > 3 {
		groups = append([]string{s[len(s)-3:]}, groups...)
		s = s[:len(s)-3]
	}
	groups = append([]string{s}, groups...)

	result := strings.Join(groups, "'")
	if neg {
		result = "-" + result
	}
	return result
}