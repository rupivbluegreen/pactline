from ai_sidecar import main


def test_main_callable() -> None:
    assert callable(main.main)
    assert callable(main.serve)
