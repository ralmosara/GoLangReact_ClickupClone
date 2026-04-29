package report

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"

	"github.com/yourorg/clickup/internal/domain"
)

// ReportMeta is the executive-style header data shown in exported PDF/XLSX.
type ReportMeta struct {
	Title         string
	WorkspaceName string
	UserName      string
	UserEmail     string
	From          time.Time
	To            time.Time
	GroupBy       GroupBy
	GeneratedAt   time.Time
	TotalTasks    int
}

func (s *Service) buildMeta(ctx context.Context, in AccomplishmentsInput, total int) ReportMeta {
	meta := ReportMeta{
		Title:       "Accomplishments Report",
		From:        in.From,
		To:          in.To,
		GroupBy:     in.GroupBy,
		GeneratedAt: time.Now(),
		TotalTasks:  total,
	}
	if s.lookups.GetWorkspace != nil {
		if w, err := s.lookups.GetWorkspace(ctx, in.WorkspaceID); err == nil && w != nil {
			meta.WorkspaceName = w.Name
		}
	}
	if s.lookups.GetUser != nil {
		if u, err := s.lookups.GetUser(ctx, in.UserID); err == nil && u != nil {
			meta.UserName = u.Name
			meta.UserEmail = u.Email
		}
	}
	return meta
}

func (s *Service) collect(ctx context.Context, in AccomplishmentsInput) ([]domain.AccomplishmentBucket, ReportMeta, error) {
	buckets, err := s.Accomplishments(ctx, in)
	if err != nil {
		return nil, ReportMeta{}, err
	}
	total := 0
	for _, b := range buckets {
		total += b.Count
	}
	return buckets, s.buildMeta(ctx, in, total), nil
}

// ExportPDF returns a printable A4 PDF with an executive-style header,
// summary, and per-bucket task lists.
func (s *Service) ExportPDF(ctx context.Context, in AccomplishmentsInput) ([]byte, error) {
	buckets, meta, err := s.collect(ctx, in)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddPage()

	// Brand bar
	pdf.SetFillColor(98, 80, 230) // brand-500-ish
	pdf.Rect(0, 0, 210, 6, "F")

	// Title
	pdf.SetY(14)
	pdf.SetFont("Arial", "B", 22)
	pdf.SetTextColor(20, 20, 28)
	pdf.Cell(0, 10, meta.Title)
	pdf.Ln(11)

	// Workspace + user line
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(80, 80, 92)
	subtitle := buildSubtitle(meta)
	if subtitle != "" {
		pdf.Cell(0, 6, subtitle)
		pdf.Ln(7)
	}

	// Period + group + generated
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(120, 120, 130)
	period := fmt.Sprintf("Period: %s  to  %s   |   Grouped by: %s   |   Generated: %s",
		meta.From.Format("Jan 2, 2006"),
		meta.To.Format("Jan 2, 2006"),
		titleGroupBy(meta.GroupBy),
		meta.GeneratedAt.Format("Jan 2, 2006 15:04"),
	)
	pdf.Cell(0, 5, period)
	pdf.Ln(8)

	// Summary card
	pdf.SetDrawColor(220, 220, 230)
	pdf.SetFillColor(248, 248, 252)
	pdf.Rect(15, pdf.GetY(), 180, 16, "FD")
	yCard := pdf.GetY()
	pdf.SetY(yCard + 3)
	pdf.SetX(20)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(140, 140, 150)
	pdf.Cell(40, 5, "TOTAL ACCOMPLISHMENTS")
	pdf.SetX(20)
	pdf.SetY(yCard + 8)
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(20, 20, 28)
	pdf.Cell(0, 6, fmt.Sprintf("%d", meta.TotalTasks))
	pdf.SetY(yCard + 16 + 6)

	if len(buckets) == 0 {
		pdf.SetFont("Arial", "I", 11)
		pdf.SetTextColor(150, 150, 160)
		pdf.Cell(0, 8, "Nothing completed in this range.")
		var buf bytes.Buffer
		if err := pdf.Output(&buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	for _, b := range buckets {
		// Bucket heading
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}
		pdf.SetFont("Arial", "B", 12)
		pdf.SetTextColor(20, 20, 28)
		pdf.Cell(0, 7, formatBucketHeading(b, meta.GroupBy))
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(140, 140, 150)
		pdf.Cell(0, 7, fmt.Sprintf("    %d %s", b.Count, pluralize(b.Count, "task", "tasks")))
		pdf.Ln(7)

		// Table header
		pdf.SetFillColor(238, 238, 244)
		pdf.SetTextColor(80, 80, 92)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(95, 7, "Task", "1", 0, "L", true, 0, "")
		pdf.CellFormat(45, 7, "List", "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 7, "Completed", "1", 0, "L", true, 0, "")
		pdf.CellFormat(10, 7, "Arc.", "1", 0, "C", true, 0, "")
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(40, 40, 50)
		for _, t := range b.Tasks {
			if pdf.GetY() > 270 {
				pdf.AddPage()
			}
			pdf.CellFormat(95, 6, truncatePDF(t.Name, 60), "1", 0, "L", false, 0, "")
			pdf.CellFormat(45, 6, truncatePDF(t.ListName, 28), "1", 0, "L", false, 0, "")
			pdf.CellFormat(30, 6, t.CompletedAt.Format("Jan 2, 15:04"), "1", 0, "L", false, 0, "")
			arc := ""
			if t.Archived {
				arc = "Yes"
			}
			pdf.CellFormat(10, 6, arc, "1", 0, "C", false, 0, "")
			pdf.Ln(-1)
		}
		pdf.Ln(3)
	}

	// Footer with workspace + page number
	pdf.AliasNbPages("")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(160, 160, 170)
		left := meta.WorkspaceName
		if left == "" {
			left = "Accomplishments report"
		}
		pdf.CellFormat(0, 5, fmt.Sprintf("%s — Page %d / {nb}", left, pdf.PageNo()), "", 0, "C", false, 0, "")
	})

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportXLSX returns an Excel workbook with a styled header, summary, and a
// flat task table (one row per task with its bucket period).
func (s *Service) ExportXLSX(ctx context.Context, in AccomplishmentsInput) ([]byte, error) {
	buckets, meta, err := s.collect(ctx, in)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Accomplishments"
	f.SetSheetName("Sheet1", sheet)

	// Column widths
	_ = f.SetColWidth(sheet, "A", "A", 38)
	_ = f.SetColWidth(sheet, "B", "B", 22)
	_ = f.SetColWidth(sheet, "C", "C", 14)
	_ = f.SetColWidth(sheet, "D", "D", 22)
	_ = f.SetColWidth(sheet, "E", "E", 10)

	// Styles
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "#14141C"},
	})
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11, Color: "#50505C"},
	})
	mutedStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#787882"},
	})
	summaryLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 9, Color: "#8C8C96"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#F5F5FA"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	summaryValueStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "#14141C"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#F5F5FA"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#6250E6"}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#4A3CB8", Style: 1},
			{Type: "right", Color: "#4A3CB8", Style: 1},
			{Type: "top", Color: "#4A3CB8", Style: 1},
			{Type: "bottom", Color: "#4A3CB8", Style: 1},
		},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#28283C"},
		Border: []excelize.Border{
			{Type: "left", Color: "#E0E0EA", Style: 1},
			{Type: "right", Color: "#E0E0EA", Style: 1},
			{Type: "top", Color: "#E0E0EA", Style: 1},
			{Type: "bottom", Color: "#E0E0EA", Style: 1},
		},
	})
	archivedStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#9090A0", Italic: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#E0E0EA", Style: 1},
			{Type: "right", Color: "#E0E0EA", Style: 1},
			{Type: "top", Color: "#E0E0EA", Style: 1},
			{Type: "bottom", Color: "#E0E0EA", Style: 1},
		},
	})

	// Header
	_ = f.SetCellValue(sheet, "A1", meta.Title)
	_ = f.MergeCell(sheet, "A1", "E1")
	_ = f.SetCellStyle(sheet, "A1", "A1", titleStyle)
	_ = f.SetRowHeight(sheet, 1, 28)

	subtitle := buildSubtitle(meta)
	if subtitle != "" {
		_ = f.SetCellValue(sheet, "A2", subtitle)
		_ = f.MergeCell(sheet, "A2", "E2")
		_ = f.SetCellStyle(sheet, "A2", "A2", subtitleStyle)
	}

	period := fmt.Sprintf("Period: %s to %s   |   Grouped by: %s   |   Generated: %s",
		meta.From.Format("Jan 2, 2006"),
		meta.To.Format("Jan 2, 2006"),
		titleGroupBy(meta.GroupBy),
		meta.GeneratedAt.Format("Jan 2, 2006 15:04"),
	)
	_ = f.SetCellValue(sheet, "A3", period)
	_ = f.MergeCell(sheet, "A3", "E3")
	_ = f.SetCellStyle(sheet, "A3", "A3", mutedStyle)

	// Summary card
	_ = f.SetCellValue(sheet, "A5", "TOTAL ACCOMPLISHMENTS")
	_ = f.MergeCell(sheet, "A5", "B5")
	_ = f.SetCellStyle(sheet, "A5", "B5", summaryLabelStyle)
	_ = f.SetCellValue(sheet, "C5", meta.TotalTasks)
	_ = f.MergeCell(sheet, "C5", "E5")
	_ = f.SetCellStyle(sheet, "C5", "E5", summaryValueStyle)
	_ = f.SetRowHeight(sheet, 5, 26)

	// Table header
	row := 7
	_ = f.SetCellValue(sheet, axisOf("A", row), "Task")
	_ = f.SetCellValue(sheet, axisOf("B", row), "List")
	_ = f.SetCellValue(sheet, axisOf("C", row), "Period")
	_ = f.SetCellValue(sheet, axisOf("D", row), "Completed at")
	_ = f.SetCellValue(sheet, axisOf("E", row), "Archived")
	_ = f.SetCellStyle(sheet, axisOf("A", row), axisOf("E", row), headerStyle)
	_ = f.SetRowHeight(sheet, row, 22)
	row++

	for _, b := range buckets {
		for _, t := range b.Tasks {
			_ = f.SetCellValue(sheet, axisOf("A", row), t.Name)
			_ = f.SetCellValue(sheet, axisOf("B", row), t.ListName)
			_ = f.SetCellValue(sheet, axisOf("C", row), b.Period)
			_ = f.SetCellValue(sheet, axisOf("D", row), t.CompletedAt.Format("2006-01-02 15:04"))
			arc := "No"
			if t.Archived {
				arc = "Yes"
			}
			_ = f.SetCellValue(sheet, axisOf("E", row), arc)
			style := cellStyle
			if t.Archived {
				style = archivedStyle
			}
			_ = f.SetCellStyle(sheet, axisOf("A", row), axisOf("E", row), style)
			row++
		}
	}

	f.SetActiveSheet(0)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// helpers --------------------------------------------------------------------

func axisOf(col string, row int) string { return fmt.Sprintf("%s%d", col, row) }

func buildSubtitle(meta ReportMeta) string {
	parts := []string{}
	if meta.WorkspaceName != "" {
		parts = append(parts, meta.WorkspaceName)
	}
	if meta.UserName != "" {
		parts = append(parts, "for "+meta.UserName)
	} else if meta.UserEmail != "" {
		parts = append(parts, "for "+meta.UserEmail)
	}
	return strings.Join(parts, "  ·  ")
}

func titleGroupBy(g GroupBy) string {
	if g == GroupByWeek {
		return "Week"
	}
	return "Day"
}

func formatBucketHeading(b domain.AccomplishmentBucket, g GroupBy) string {
	if g == GroupByWeek {
		end := b.EndsAt.Add(-24 * time.Hour)
		return fmt.Sprintf("Week of %s – %s",
			b.StartsAt.Format("Jan 2"), end.Format("Jan 2, 2006"))
	}
	return b.StartsAt.Format("Monday, Jan 2, 2006")
}

func pluralize(n int, single, plural string) string {
	if n == 1 {
		return single
	}
	return plural
}

func truncatePDF(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes-1]) + "…"
}

