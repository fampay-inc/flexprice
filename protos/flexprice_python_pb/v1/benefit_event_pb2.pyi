from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class BenefitEvent(_message.Message):
    __slots__ = ["event_id", "username", "subscription_id", "feature_id", "category", "value", "timestamp", "benefit_type"]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    FEATURE_ID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    BENEFIT_TYPE_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    username: str
    subscription_id: str
    feature_id: str
    category: str
    value: int
    timestamp: int
    benefit_type: str
    def __init__(self, event_id: _Optional[str] = ..., username: _Optional[str] = ..., subscription_id: _Optional[str] = ..., feature_id: _Optional[str] = ..., category: _Optional[str] = ..., value: _Optional[int] = ..., timestamp: _Optional[int] = ..., benefit_type: _Optional[str] = ...) -> None: ...

class BenefitReversalEvent(_message.Message):
    __slots__ = ["event_id", "username", "value", "timestamp", "original_event_id"]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    username: str
    value: int
    timestamp: int
    original_event_id: str
    def __init__(self, event_id: _Optional[str] = ..., username: _Optional[str] = ..., value: _Optional[int] = ..., timestamp: _Optional[int] = ..., original_event_id: _Optional[str] = ...) -> None: ...
