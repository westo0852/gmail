#!/usr/bin/env bash
set -euo pipefail

echo -n 'Generating encrypted PDFs from Markdown files with pandoc and qpdf...'
for file in *.md; do
  if [[ "$file" == "README.md" ]]; then continue; fi
  pandoc "$file" -o "${file%.md}.pdf"
  qpdf --encrypt "$USER_PASS" "$OWNER_PASS" 256 \
    -- "${file%.md}.pdf" "${file%.md}_encrypted.pdf"
  rm "${file%.md}.pdf"
done
echo 'Done!'

echo 'Running main.go to send PDFs as email attachments...'
go run main.go --attachments \
  $(find . -type f -name '*_encrypted.pdf' | tr '\n' ' ')
