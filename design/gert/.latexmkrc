# latexmk configuration for gert-v2 design docs.
# Uses biber for biblatex bibliography processing and enables shell-escape
# for minted syntax highlighting.

$pdf_mode = 1;
$bibtex_use = 2;  # use biber when biblatex is detected (reads .bcf file)
$jobname = 'gert';
$out_dir = '.';  # output gert.pdf next to main.tex
