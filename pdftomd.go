package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
)

func main() {
	// 1. Vérification des arguments
	// os.Args[0] est le nom du programme, [1] le PDF, [2] le Markdown
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <input.pdf> <output.md>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	fmt.Printf("Conversion de '%s' vers '%s'...\n", inputPath, outputPath)

	// 2. Lancer la conversion
	err := convertPDFToMarkdown(inputPath, outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur critique : %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Conversion terminée avec succès !")
}

func convertPDFToMarkdown(pdfPath, outputPath string) error {
	// Ouvrir le fichier PDF
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le PDF : %w", err)
	}
	defer f.Close()

	var buffer bytes.Buffer
	totalPage := r.NumPage()

	// Boucle sur toutes les pages
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		// Extraction du texte
		text, err := p.GetPlainText(nil)
		if err != nil {
			fmt.Printf("Avertissement : impossible de lire la page %d\n", pageIndex)
			continue
		}

		// Formatage Markdown
		buffer.WriteString(fmt.Sprintf("\n# Page %d\n", pageIndex))
		buffer.WriteString(text)
		buffer.WriteString("\n\n---\n") // Séparateur horizontal
	}

	// Sauvegarde dans le fichier final
	err = os.WriteFile(outputPath, buffer.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("impossible d'écrire le fichier Markdown : %w", err)
	}

	return nil
}