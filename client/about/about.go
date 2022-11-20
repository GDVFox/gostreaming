package about

import "github.com/pterm/pterm"

// HandleAbout выводит сообщение с помощью.
func HandleAbout() {
	title, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("Go", pterm.NewStyle(pterm.FgLightMagenta)),
		pterm.NewLettersFromStringWithStyle("Streaming", pterm.NewStyle(pterm.FgCyan))).
		Srender()

	pterm.DefaultCenter.Println(title)
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		"Distributed dataflow managing tool\n" +
			"by Daniil Gavrilovsky.\n" +
			"GitHub repo: 'https://github.com/GDVFox/gostreaming'\n" +
			"2022")
}
