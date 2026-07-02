package semantic

func PanelButtonWriteFields() []string {
	return cloneStrings([]string{
		FieldID,
		FieldDeviceID,
		FieldName,
		FieldAlias,
		FieldKeyValue,
		FieldIndex,
		InternalField(DomainPanel, FieldTargetID),
		InternalField(DomainPanel, FieldTargetType),
		FieldVisible,
		FieldIcon,
		FieldSort,
		FieldType,
		FieldExtend,
	})
}

func KnobDetailWriteFields() []string {
	return cloneStrings([]string{
		FieldID,
		FieldIndex,
		FieldConfigType,
		FieldMode,
		FieldModel,
		InternalField(DomainKnob, FieldTargetID),
		InternalField(DomainKnob, FieldTargetType),
		InternalField(DomainPanel, FieldTargetType),
		InternalField(DomainKnob, FieldSubIndex),
		InternalField(DomainPanel, FieldTargetName),
		InternalKnobActionParamsField(),
		InternalField(DomainKnob, FieldSensitivity),
		FieldAction,
		FieldProperty,
		FieldValue,
		FieldDetails,
	})
}

func PanelEventDetailWriteFields() []string {
	return cloneStrings([]string{
		FieldID,
		FieldRoomID,
		InternalField(DomainPanel, FieldTargetID),
		InternalField(DomainKnob, FieldTargetType),
		InternalField(DomainPanel, FieldTargetType),
		InternalField(DomainPanel, FieldSubIndex),
		InternalPanelActionParamsField(),
		FieldRank,
		InternalField(DomainPanel, FieldTargetName),
		InternalRepeatTypeField(),
		InternalRepeatValueField(),
		FieldStartTime,
		FieldEndTime,
		FieldAction,
		FieldProperty,
		FieldValue,
		FieldDelay,
		FieldDuration,
	})
}

func cloneStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}
