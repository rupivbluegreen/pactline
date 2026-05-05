from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class ParseRequest(_message.Message):
    __slots__ = ("document_id", "content", "mime_type")
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    document_id: str
    content: bytes
    mime_type: str
    def __init__(self, document_id: _Optional[str] = ..., content: _Optional[bytes] = ..., mime_type: _Optional[str] = ...) -> None: ...

class ParseResponse(_message.Message):
    __slots__ = ("text", "page_count", "segments")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SEGMENTS_FIELD_NUMBER: _ClassVar[int]
    text: str
    page_count: int
    segments: _containers.RepeatedCompositeFieldContainer[TextSegment]
    def __init__(self, text: _Optional[str] = ..., page_count: _Optional[int] = ..., segments: _Optional[_Iterable[_Union[TextSegment, _Mapping]]] = ...) -> None: ...

class TextSegment(_message.Message):
    __slots__ = ("locator", "char_start", "char_end")
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    CHAR_START_FIELD_NUMBER: _ClassVar[int]
    CHAR_END_FIELD_NUMBER: _ClassVar[int]
    locator: str
    char_start: int
    char_end: int
    def __init__(self, locator: _Optional[str] = ..., char_start: _Optional[int] = ..., char_end: _Optional[int] = ...) -> None: ...
