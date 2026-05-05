from pathlib import Path

from document_sidecar.handlers.parse import PDF_MIME, parse_document

FIXTURE = Path(__file__).parent / "fixtures" / "sample.pdf"


def test_parse_pdf_text_and_segments() -> None:
    body = FIXTURE.read_bytes()
    resp = parse_document(body, PDF_MIME)

    assert resp.page_count == 2
    assert "MUTUAL NDA" in resp.text
    assert "Acme Corp" in resp.text
    assert "Governing Law" in resp.text

    locators = [s.locator for s in resp.segments]
    assert locators == ["page:1", "page:2"]

    cursor = 0
    for s in resp.segments:
        assert s.char_start == cursor
        assert s.char_end >= s.char_start
        cursor = s.char_end
    assert cursor == len(resp.text)
