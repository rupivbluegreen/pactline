import document_pb2 as _document_pb2
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

class ExtractRequest(_message.Message):
    __slots__ = ("document_id", "parsed_text", "segments", "field_names", "model_id", "prompt_version")
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    PARSED_TEXT_FIELD_NUMBER: _ClassVar[int]
    SEGMENTS_FIELD_NUMBER: _ClassVar[int]
    FIELD_NAMES_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    PROMPT_VERSION_FIELD_NUMBER: _ClassVar[int]
    document_id: str
    parsed_text: str
    segments: _containers.RepeatedCompositeFieldContainer[_document_pb2.TextSegment]
    field_names: _containers.RepeatedScalarFieldContainer[str]
    model_id: str
    prompt_version: str
    def __init__(self, document_id: _Optional[str] = ..., parsed_text: _Optional[str] = ..., segments: _Optional[_Iterable[_Union[_document_pb2.TextSegment, _Mapping]]] = ..., field_names: _Optional[_Iterable[str]] = ..., model_id: _Optional[str] = ..., prompt_version: _Optional[str] = ...) -> None: ...

class ExtractResponse(_message.Message):
    __slots__ = ("fields",)
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[ExtractedField]
    def __init__(self, fields: _Optional[_Iterable[_Union[ExtractedField, _Mapping]]] = ...) -> None: ...

class ExtractedField(_message.Message):
    __slots__ = ("field_name", "value", "value_json", "citation")
    FIELD_NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    VALUE_JSON_FIELD_NUMBER: _ClassVar[int]
    CITATION_FIELD_NUMBER: _ClassVar[int]
    field_name: str
    value: str
    value_json: str
    citation: Citation
    def __init__(self, field_name: _Optional[str] = ..., value: _Optional[str] = ..., value_json: _Optional[str] = ..., citation: _Optional[_Union[Citation, _Mapping]] = ...) -> None: ...

class Citation(_message.Message):
    __slots__ = ("document_id", "locator", "span_start", "span_end", "model_id", "prompt_version", "timestamp")
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    SPAN_START_FIELD_NUMBER: _ClassVar[int]
    SPAN_END_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    PROMPT_VERSION_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    document_id: str
    locator: str
    span_start: int
    span_end: int
    model_id: str
    prompt_version: str
    timestamp: str
    def __init__(self, document_id: _Optional[str] = ..., locator: _Optional[str] = ..., span_start: _Optional[int] = ..., span_end: _Optional[int] = ..., model_id: _Optional[str] = ..., prompt_version: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...
