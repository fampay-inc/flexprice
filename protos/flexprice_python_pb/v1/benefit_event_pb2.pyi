from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class BenefitEvent(_message.Message):
    __slots__ = ["event_id", "subscription_id", "cycle_id", "feature_id", "category", "value", "timestamp"]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    CYCLE_ID_FIELD_NUMBER: _ClassVar[int]
    FEATURE_ID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    subscription_id: str
    cycle_id: str
    feature_id: str
    category: str
    value: int
    timestamp: int
    def __init__(self, event_id: _Optional[str] = ..., subscription_id: _Optional[str] = ..., cycle_id: _Optional[str] = ..., feature_id: _Optional[str] = ..., category: _Optional[str] = ..., value: _Optional[int] = ..., timestamp: _Optional[int] = ...) -> None: ...
