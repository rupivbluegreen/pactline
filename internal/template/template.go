// Package template is the in-process templating engine for non-DOCX
// output (email subjects/bodies, webhook payloads, etc.).
//
// Phase 0: empty. Phase 1: thin wrapper over text/template.
//
// DOCX rendering is intentionally NOT here — it lives in the Python
// document sidecar (PyMuPDF / python-docx / LibreOffice headless).
package template
