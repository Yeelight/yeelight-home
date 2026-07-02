package semantic

type ResponseFieldMapping struct {
	Public   string
	Internal []string
}

const (
	internalRows                   = "rows"
	internalList                   = "list"
	internalRecords                = "records"
	internalContent                = "content"
	internalResults                = "results"
	internalDataList               = "dataList"
	internalPage                   = "page"
	internalSlots                  = "slots"
	internalHouses                 = "houses"
	internalHouseList              = "houseList"
	internalHouseListSnake         = "house_list"
	internalUserScenes             = "userscenes"
	internalMeshGroupsLower        = "meshgroups"
	internalMeshGroups             = "meshGroups"
	internalMeshGroupID            = "meshGroupId"
	internalConfigs                = "configs"
	internalSupportedBridge        = "supportedBridgeType"
	internalGatewayName            = "gatewayName"
	internalDeviceName             = "deviceName"
	internalRoomName               = "roomName"
	internalGroupName              = "groupName"
	internalAreaName               = "areaName"
	internalAutomationID           = "automationId"
	internalAutomationName         = "automationName"
	internalUserGroupID            = "userGroupId"
	internalTypoGroupName          = "nane"
	internalFirmwareShort          = "fwVersion"
	internalProductCategoryIDLower = "pcid"
	internalRemark                 = "remark"
	internalPosition               = "position"
	internalEnable                 = "enable"
	internalOnlineFlag             = "isOnline"
	internalAccountID              = "accountId"
	internalNickname               = "nickname"
	internalNickName               = "nickName"
	internalMemberRole             = "memberRole"
	internalMemberList             = "memberList"
	internalPhone                  = "phone"
	internalPhoneNumber            = "phoneNumber"
	internalMobile                 = "mobile"
	internalMobilePhone            = "mobilePhone"
	internalEmail                  = "email"
	internalMail                   = "mail"
	internalSubDevices             = "subDevices"
	internalScheduleJobID          = "scheduleJobId"
	internalScheduleJobs           = "scheduleJobs"
	internalMessageID              = "messageId"
	internalMessageType            = "messageType"
	internalReadStatus             = "readStatus"
	internalReadFlag               = "isRead"
	internalBizType                = "bizType"
	internalBizID                  = "bizId"
	internalQuestion               = "question"
	internalFAQType                = "faqType"
	internalFAQItems               = "faqItems"
	internalLabel                  = "label"
	internalEnglishDescription     = "englishDescription"
	internalEnglishName            = "enName"
	internalLanguageCode           = "languageCode"
	internalLanguageName           = "languageName"
	internalLocalName              = "localName"
	internalDomainID               = "domainId"
	internalDomainName             = "domainName"
	internalDomainCode             = "domainCode"
	internalCategoryID             = "categoryId"
	internalCategoryName           = "categoryName"
	internalCategoryCode           = "categoryCode"
	internalComponentCode          = "componentCode"
	internalComponentType          = "componentType"
	internalProps                  = "props"
	internalPropertyID             = "propertyId"
	internalPropertyName           = "propertyName"
	internalPropertyCode           = "propertyCode"
	internalIdentifier             = "identifier"
	internalPropertyIDShort        = "propId"
	internalActionName             = "actionName"
	internalEventType              = "eventType"
	internalSKUMaterialCode        = "skuMaterialCode"
	internalSKU                    = "sku"
	internalSPU                    = "spu"
	internalUpdateTime             = "updateTime"
	internalClearAll               = "clearAll"
	internalOverwrite              = "overwrite"
	internalQueryString            = "queryString"
	internalProductSkuFullText     = "productSkuFullText"
	internalBrand                  = "brand"
	internalProductStatus          = "productStatus"
	internalImageShort             = "img"
	internalRoomCount              = "roomNum"
	internalDeviceCount            = "deviceNum"
	internalUnboundDeviceCount     = "unbindDeviceNum"
	internalGatewayCount           = "gatewayNum"
	internalUnboundGatewayCount    = "unbindGatewayNum"
	internalSceneCount             = "sceneNum"
	internalAutomationCount        = "automationNum"
	internalAreaCount              = "areaNum"
)

var (
	responseRowsContainers                  = []string{internalRows, internalList, FieldData, internalRecords, FieldItems, FieldResult, internalPage, internalHouses, internalHouseList, internalHouseListSnake, internalContent, internalResults, internalDataList}
	sceneRowContainers                      = []string{internalList, internalRows, FieldScenes, internalUserScenes}
	automationRowContainers                 = []string{internalList, internalRows, FieldAutomations}
	gatewayRowContainers                    = []string{FieldGateways, internalRows, internalList}
	deviceRowContainers                     = []string{FieldDevices, internalRows, internalList}
	meshGroupRowContainers                  = []string{internalMeshGroupsLower, internalMeshGroups}
	configRowContainers                     = []string{internalRows, internalList}
	deviceSchemaRowContainers               = []string{FieldDevices, internalRows, internalList}
	scheduleJobRowContainers                = []string{internalList, internalRows, internalScheduleJobs}
	memberRowContainers                     = []string{internalRows, internalList, FieldMembers, internalMemberList, FieldData}
	naturalLightingDesignFields             = []string{FieldRooms, internalRoomList, FieldItems, internalSlots, FieldDevices, FieldGroups, FieldScenes, FieldAutomations}
	importCleanupFields                     = []string{FieldRooms, internalRoomList, FieldDevices, internalDeviceList, FieldDeviceSlots, FieldGroups, internalGroupList, FieldScenes, FieldAutomations}
	importOverwriteFields                   = []string{internalClearAll, internalOverwrite}
	responseIDFields                        = []string{FieldID, FieldHouseID, FieldRoomID, FieldSceneID}
	sensorEventForwardFields                = []string{FieldDeviceID, FieldSensorID, FieldEventID, FieldName, FieldStatus, FieldValid}
	weatherQueryForwardFields               = []string{FieldArea, FieldDimension, FieldTimeStart, FieldTimeEnd, FieldDate, FieldLanguage}
	houseSummaryIconFields                  = []string{FieldIcon, internalImageShort}
	houseSummaryDescriptionFields           = []string{internalDescription, FieldDescription}
	houseSummaryPresenceFields              = []string{FieldAreaCode, FieldAreaName, FieldIcon, internalImageShort, internalDescription, FieldDescription}
	houseSummaryCountFields                 = []string{internalRoomCount, internalDeviceCount, internalUnboundDeviceCount, internalGatewayCount, internalUnboundGatewayCount, internalSceneCount, internalAutomationCount, internalAreaCount}
	houseSummaryCountMappings               = []ResponseFieldMapping{{FieldRooms, []string{internalRoomCount}}, {FieldDevices, []string{internalDeviceCount}}, {FieldUnboundDevices, []string{internalUnboundDeviceCount}}, {FieldGateways, []string{internalGatewayCount}}, {FieldUnboundGateways, []string{internalUnboundGatewayCount}}, {FieldScenes, []string{internalSceneCount}}, {FieldAutomations, []string{internalAutomationCount}}, {FieldAreas, []string{internalAreaCount}}}
	entitySummaryIDFields                   = []string{FieldID, FieldHouseID, FieldAreaID, FieldRoomID, FieldGroupID, FieldSceneID, internalAutomationID, FieldDeviceID}
	entitySummaryNameFields                 = []string{FieldName, FieldHouseName, internalAreaName, internalRoomName, internalGroupName, FieldSceneName, internalAutomationName}
	entitySummaryHouseIDFields              = []string{FieldHouseID}
	entitySummaryRoomIDFields               = []string{FieldRoomID}
	entitySummaryStatusFields               = []string{FieldStatus}
	entitySummaryOnlineFields               = []string{FieldOnline, internalOnlineFlag}
	entitySummaryBindFields                 = []string{FieldBind, internalBindFlag}
	entitySummaryVirtualFields              = []string{FieldVirtual, internalVirtualFlag}
	sceneSummaryMappings                    = []ResponseFieldMapping{{FieldID, []string{FieldSceneID, FieldID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldRoomID, []string{FieldRoomID}}, {FieldGatewayDeviceID, []string{FieldGatewayDeviceID}}, {FieldName, []string{FieldName}}, {FieldImage, []string{internalImage}}, {FieldSequence, []string{internalSequence}}, {FieldRoomRank, []string{FieldRoomRank}}, {FieldTimeInterval, []string{FieldTimeInterval}}}
	automationSummaryMappings               = []ResponseFieldMapping{{FieldID, []string{FieldID, internalAutomationID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldName, []string{FieldName}}, {FieldStatus, []string{FieldStatus}}, {FieldVersion, []string{FieldVersion}}, {FieldRuleID, []string{FieldRuleID}}}
	sortedDeviceSummaryMappings             = []ResponseFieldMapping{{FieldID, []string{FieldDeviceID, FieldID}}, {FieldDeviceIdentifier, []string{internalDeviceIdentifier}}, {FieldGatewayDeviceID, []string{FieldGatewayDeviceID}}, {FieldCapabilityProductID, []string{internalProductID}}, {FieldType, []string{FieldType}}, {FieldName, []string{FieldAlias, FieldName}}, {FieldImage, []string{internalImage}}, {FieldCapability, []string{FieldCapability}}, {FieldBind, []string{internalBindFlag}}, {FieldRoomID, []string{FieldRoomID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldVirtual, []string{internalVirtualFlag}}, {FieldIndex, []string{FieldIndex}}, {FieldRank, []string{FieldRank}}}
	configSummaryMappings                   = []ResponseFieldMapping{{FieldProperty, []string{FieldProperty, internalPropertyName, internalPropertyCode, internalPropertyIDShort}}, {FieldDescription, []string{FieldDescription, internalDescription}}, {FieldAccess, []string{FieldAccess}}, {FieldFormat, []string{FieldFormat}}}
	gatewayStringMappings                   = []ResponseFieldMapping{{FieldID, []string{FieldID, FieldGatewayID, FieldDeviceID}}, {FieldDeviceIdentifier, []string{internalDeviceIdentifier}}, {FieldGatewayDeviceID, []string{FieldGatewayDeviceID}}, {FieldCapabilityProductID, []string{internalProductID}}, {FieldProductCategoryID, []string{internalProductCategoryID, internalProductCategoryIDLower}}, {FieldType, []string{FieldType}}, {FieldName, []string{FieldName, internalGatewayName, internalDeviceName, FieldAlias, internalRemark}}, {FieldImage, []string{internalImage}}, {FieldHouseID, []string{FieldHouseID}}, {FieldRoomID, []string{FieldRoomID}}, {FieldRoomCount, []string{internalRoomCount}}, {FieldDeviceCount, []string{internalDeviceCount}}, {FieldCapability, []string{FieldCapability}}, {FieldConnectType, []string{FieldConnectType}}, {FieldTypeName, []string{FieldTypeName}}, {FieldModel, []string{FieldModel}}, {FieldFirmwareVersion, []string{FieldFirmwareVersion, internalFirmwareShort, FieldVersion}}}
	gatewayBoolMappings                     = []ResponseFieldMapping{{FieldOnline, []string{FieldOnline, internalOnlineFlag}}, {FieldBind, []string{FieldBind, internalBindFlag}}, {FieldEnabled, []string{FieldEnabled, internalEnable}}}
	areaSummaryMappings                     = []ResponseFieldMapping{{FieldID, []string{FieldAreaID, FieldID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldName, []string{FieldName, internalAreaName}}, {FieldDescription, []string{FieldDescription, internalDescription}}, {FieldIcon, []string{FieldIcon}}, {FieldImage, []string{internalImage}}, {FieldLevel, []string{FieldLevel}}, {FieldParentID, []string{FieldParentID}}, {FieldRank, []string{FieldRank}}, {FieldDeviceCount, []string{FieldDeviceCount, internalDeviceCount}}, {FieldRoomCount, []string{FieldRoomCount, internalRoomCount}}}
	roomSummaryMappings                     = []ResponseFieldMapping{{FieldID, []string{FieldRoomID, FieldID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldName, []string{FieldName}}, {FieldCapability, []string{FieldCapability}}, {FieldImage, []string{internalImage, FieldIcon}}, {FieldGatewayDeviceID, []string{FieldGatewayDeviceID}}, {FieldSequence, []string{internalSequence}}, {FieldRank, []string{FieldRank}}}
	groupSummaryMappings                    = []ResponseFieldMapping{{FieldID, []string{internalUserGroupID, FieldGroupID, FieldID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldName, []string{FieldName, internalTypoGroupName, internalGroupName}}, {FieldRank, []string{FieldRank}}, {FieldIcon, []string{FieldIcon}}, {FieldImage, []string{internalImage}}}
	deviceSummaryMappings                   = []ResponseFieldMapping{{FieldID, []string{FieldDeviceID, FieldID}}, {FieldDeviceIdentifier, []string{internalDeviceIdentifier}}, {FieldGatewayDeviceID, []string{FieldGatewayDeviceID}}, {FieldCapabilityProductID, []string{internalProductID}}, {FieldType, []string{FieldType}}, {FieldName, []string{FieldName, FieldAlias, internalRemark}}, {FieldImage, []string{internalImage}}, {FieldSequence, []string{internalSequence}}, {FieldPosition, []string{internalPosition}}, {FieldHouseID, []string{FieldHouseID}}, {FieldRoomID, []string{FieldRoomID}}, {FieldCapability, []string{FieldCapability}}, {FieldRoomRank, []string{FieldRoomRank}}, {FieldBind, []string{internalBindFlag}}, {FieldVirtual, []string{internalVirtualFlag}}, {FieldConnectType, []string{FieldConnectType}}, {FieldTypeName, []string{FieldTypeName}}}
	meshGroupSummaryMappings                = []ResponseFieldMapping{{FieldID, []string{internalMeshGroupID, FieldMeshGroupID, FieldGroupID, FieldID}}, {FieldHouseID, []string{FieldHouseID}}, {FieldName, []string{FieldName, internalGroupName}}, {FieldRoomID, []string{FieldRoomID}}}
	deviceCapabilityProductIDFields         = []string{internalProductID, FieldCapabilityProductID}
	deviceCapabilityProductCategoryIDFields = []string{internalProductCategoryID, FieldProductCategoryID}
	componentIDFields                       = []string{internalCloudComponentID, FieldComponentID}
	propertyIDFields                        = []string{internalPropertyIDShort, FieldID, internalPropertyID}
	descriptionFields                       = []string{internalDescription, FieldDescription}
	actionIDFields                          = []string{internalActionName, FieldID, FieldName}
	accountIDFields                         = []string{FieldID, FieldUID, FieldUserID, internalAccountID}
	memberIDFields                          = []string{FieldUID, FieldUserID, FieldMemberID, FieldID, internalAccountID}
	accountDisplayNameFields                = []string{internalNickname, internalNickName, FieldName, FieldDisplayName, internalRemark}
	accountPhoneFields                      = []string{internalPhone, internalPhoneNumber, internalMobile, internalMobilePhone}
	accountEmailFields                      = []string{internalEmail, internalMail}
	memberRoleFields                        = []string{FieldRole, FieldUserRole, internalMemberRole}
	productCodeFields                       = []string{internalProductCode, internalSKUMaterialCode, FieldProductCode}
	productSKUFields                        = []string{FieldProductSKU, internalSKU}
	productSPUFields                        = []string{FieldProductSPU, internalSPU}
	productPediaQueryFields                 = []string{FieldMultiField, FieldKeyword, FieldQuery, internalQueryString, FieldName, FieldProductName, FieldProductShortName, FieldProductCode, internalProductCode, FieldSKU, FieldProductSKU, internalProductSkuFullText, FieldSPU, FieldProductSPU, FieldModel, FieldProductModel, FieldModelNo, FieldBarcode, FieldCapabilityProductID, internalProductID}
	scheduleJobIDFields                     = []string{FieldID, internalScheduleJobID}
	messageIDFields                         = []string{FieldID, internalMessageID}
	messageTypeFields                       = []string{FieldType, internalMessageType}
	messageStatusFields                     = []string{FieldStatus, internalReadStatus, internalReadFlag}
	messageTargetTypeFields                 = []string{FieldTargetType, internalBizType}
	messageTargetIDFields                   = []string{FieldTargetID, internalBizID}
	messageContentFields                    = []string{internalContent, FieldSummary, FieldMessage}
	productDomainIDFields                   = []string{FieldID, internalDomainID}
	productDomainNameFields                 = []string{FieldName, internalDomainName}
	productDomainCodeFields                 = []string{FieldCode, internalDomainCode}
	faqTitleFields                          = []string{FieldTitle, internalQuestion, FieldName}
	faqTypeFields                           = []string{FieldType, internalFAQType}
	localeCodeFields                        = []string{FieldCode, internalLanguageCode, FieldLocale}
	localeNameFields                        = []string{FieldName, FieldDescription, internalLanguageName}
	nativeNameFields                        = []string{FieldNativeName, internalLocalName}
	codeDescriptionFields                   = []string{FieldDescription, FieldName, internalLabel}
	englishDescriptionFields                = []string{FieldEnglishDescription, internalEnglishDescription, internalEnglishName}
	thingCategoryIDFields                   = []string{FieldID, internalCloudComponentID, internalCategoryID}
	thingCategoryNameFields                 = []string{FieldName, internalCategoryName}
	thingCategoryCodeFields                 = []string{FieldCode, internalCategoryCode}
	thingComponentCodeFields                = []string{FieldCode, internalComponentCode}
	thingComponentTypeFields                = []string{FieldType, internalComponentType}
	thingPropertyIDFields                   = []string{FieldID, internalPropertyID}
	thingPropertyNameFields                 = []string{FieldName, internalPropertyName}
	thingPropertyCodeFields                 = []string{FieldCode, internalPropertyCode, internalIdentifier}
	thingEventTypeFields                    = []string{internalEventType, FieldEventTypeID, FieldType}
	productPediaSummaryMappings             = []ResponseFieldMapping{{FieldID, []string{FieldID}}, {FieldProductCode, []string{internalProductCode, FieldProductCode}}, {FieldCapabilityProductID, []string{internalProductID, FieldCapabilityProductID, FieldID}}, {FieldProductCategoryID, []string{internalProductCategoryID, FieldProductCategoryID}}, {FieldProductName, []string{FieldProductName, FieldName}}, {FieldProductBrand, []string{FieldProductBrand, internalBrand}}, {FieldProductModel, []string{FieldProductModel, FieldModel}}, {FieldProductSKU, []string{FieldProductSKU, internalSKU}}, {FieldProductSPU, []string{FieldProductSPU, internalSPU}}, {FieldProductLine, []string{FieldProductLine}}, {FieldProductCategory, []string{FieldProductCategory, internalCategoryName}}, {FieldProductLargeClass, []string{FieldProductLargeClass}}, {FieldProductSmallClass, []string{FieldProductSmallClass}}, {FieldProductShortName, []string{FieldProductShortName}}, {FieldProductSeries, []string{FieldProductSeries}}, {FieldBarcode, []string{FieldBarcode}}, {FieldModelNo, []string{FieldModelNo}}, {FieldBaseUnit, []string{FieldBaseUnit}}, {FieldProductDeclareNo, []string{FieldProductDeclareNo}}, {FieldProductDeclareName, []string{FieldProductDeclareName}}, {FieldProductDeclareUnit, []string{FieldProductDeclareUnit}}, {FieldProductStatusName, []string{FieldProductStatusName, internalProductStatus}}, {FieldProductSaleType, []string{FieldProductSaleType}}, {FieldQuotationType, []string{FieldQuotationType}}, {FieldProductTypeName, []string{FieldProductTypeName}}, {FieldSupportYeelightPro, []string{FieldSupportYeelightPro}}, {FieldSupportHomeKit, []string{FieldSupportHomeKit}}, {FieldPediaDisplay, []string{FieldPediaDisplay}}}
	productPediaAttachmentMappings          = []ResponseFieldMapping{{FieldID, []string{FieldID}}, {FieldTargetID, []string{internalBizID}}, {FieldTargetType, []string{internalBizType}}, {FieldURL, []string{FieldURL}}, {FieldType, []string{FieldType}}, {FieldName, []string{FieldName}}, {FieldRank, []string{FieldRank, FieldSort, FieldOrder}}, {FieldCreatedAt, []string{FieldCreateTime, FieldCreatedAt}}, {FieldUpdatedAt, []string{internalUpdateTime, FieldUpdatedAt}}}
)

func ResponseRowsContainers() []string { return cloneStrings(responseRowsContainers) }
func SceneRowContainers() []string     { return cloneStrings(sceneRowContainers) }
func AutomationRowContainers() []string {
	return cloneStrings(automationRowContainers)
}
func GatewayRowContainers() []string   { return cloneStrings(gatewayRowContainers) }
func DeviceRowContainers() []string    { return cloneStrings(deviceRowContainers) }
func MeshGroupRowContainers() []string { return cloneStrings(meshGroupRowContainers) }
func ConfigRowContainers() []string    { return cloneStrings(configRowContainers) }
func DeviceSchemaRowContainers() []string {
	return cloneStrings(deviceSchemaRowContainers)
}
func ScheduleJobRowContainers() []string {
	return cloneStrings(scheduleJobRowContainers)
}
func MemberRowContainers() []string { return cloneStrings(memberRowContainers) }
func NaturalLightingDesignFields() []string {
	return cloneStrings(naturalLightingDesignFields)
}
func ImportCleanupFields() []string      { return cloneStrings(importCleanupFields) }
func ImportOverwriteFields() []string    { return cloneStrings(importOverwriteFields) }
func ResponseIDFields() []string         { return cloneStrings(responseIDFields) }
func SensorEventForwardFields() []string { return cloneStrings(sensorEventForwardFields) }
func WeatherQueryForwardFields() []string {
	return cloneStrings(weatherQueryForwardFields)
}
func HouseSummaryIconFields() []string        { return cloneStrings(houseSummaryIconFields) }
func HouseSummaryDescriptionFields() []string { return cloneStrings(houseSummaryDescriptionFields) }
func HouseSummaryPresenceFields() []string    { return cloneStrings(houseSummaryPresenceFields) }
func HouseSummaryCountFields() []string       { return cloneStrings(houseSummaryCountFields) }
func HouseSummaryCountMappings() []ResponseFieldMapping {
	return cloneResponseMappings(houseSummaryCountMappings)
}

func EntitySummaryIDFields() []string { return cloneStrings(entitySummaryIDFields) }
func EntitySummaryIDFieldsForType(entityType string) []string {
	switch entityType {
	case "device":
		return []string{FieldDeviceID, FieldID}
	case "gateway":
		return []string{FieldGatewayID, FieldDeviceID, FieldID}
	case "area":
		return []string{FieldAreaID, FieldID}
	case "room":
		return []string{FieldRoomID, FieldID}
	case "group":
		return []string{FieldGroupID, internalUserGroupID, internalMeshGroupID, FieldMeshGroupID, FieldID}
	case "scene":
		return []string{FieldSceneID, FieldID}
	case "automation":
		return []string{FieldID, internalAutomationID, FieldRuleID}
	case "home":
		return []string{FieldHouseID, FieldID}
	default:
		return EntitySummaryIDFields()
	}
}
func EntitySummaryNameFields() []string    { return cloneStrings(entitySummaryNameFields) }
func EntitySummaryHouseIDFields() []string { return cloneStrings(entitySummaryHouseIDFields) }
func EntitySummaryRoomIDFields() []string  { return cloneStrings(entitySummaryRoomIDFields) }
func EntitySummaryStatusFields() []string  { return cloneStrings(entitySummaryStatusFields) }
func EntitySummaryOnlineFields() []string  { return cloneStrings(entitySummaryOnlineFields) }
func EntitySummaryBindFields() []string    { return cloneStrings(entitySummaryBindFields) }
func EntitySummaryVirtualFields() []string { return cloneStrings(entitySummaryVirtualFields) }

func SceneSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(sceneSummaryMappings)
}
func AutomationSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(automationSummaryMappings)
}
func SortedDeviceSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(sortedDeviceSummaryMappings)
}
func ConfigSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(configSummaryMappings)
}
func GatewayStringMappings() []ResponseFieldMapping {
	return cloneResponseMappings(gatewayStringMappings)
}
func GatewayBoolMappings() []ResponseFieldMapping { return cloneResponseMappings(gatewayBoolMappings) }
func AreaSummaryMappings() []ResponseFieldMapping { return cloneResponseMappings(areaSummaryMappings) }
func RoomSummaryMappings() []ResponseFieldMapping { return cloneResponseMappings(roomSummaryMappings) }
func GroupSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(groupSummaryMappings)
}
func DeviceSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(deviceSummaryMappings)
}
func MeshGroupSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(meshGroupSummaryMappings)
}
func ProductPediaAttachmentMappings() []ResponseFieldMapping {
	return cloneResponseMappings(productPediaAttachmentMappings)
}
func ProductPediaSummaryMappings() []ResponseFieldMapping {
	return cloneResponseMappings(productPediaSummaryMappings)
}

func DeviceCapabilityProductIDFields() []string {
	return cloneStrings(deviceCapabilityProductIDFields)
}
func DeviceCapabilityProductCategoryIDFields() []string {
	return cloneStrings(deviceCapabilityProductCategoryIDFields)
}
func ComponentIDFields() []string        { return cloneStrings(componentIDFields) }
func PropertyIDFields() []string         { return cloneStrings(propertyIDFields) }
func DescriptionFields() []string        { return cloneStrings(descriptionFields) }
func ActionIDFields() []string           { return cloneStrings(actionIDFields) }
func AccountIDFields() []string          { return cloneStrings(accountIDFields) }
func MemberIDFields() []string           { return cloneStrings(memberIDFields) }
func AccountDisplayNameFields() []string { return cloneStrings(accountDisplayNameFields) }
func AccountPhoneFields() []string       { return cloneStrings(accountPhoneFields) }
func AccountEmailFields() []string       { return cloneStrings(accountEmailFields) }
func MemberRoleFields() []string         { return cloneStrings(memberRoleFields) }
func ProductCodeFields() []string        { return cloneStrings(productCodeFields) }
func ProductSKUFields() []string         { return cloneStrings(productSKUFields) }
func ProductSPUFields() []string         { return cloneStrings(productSPUFields) }
func ProductPediaQueryFields() []string  { return cloneStrings(productPediaQueryFields) }
func ScheduleJobIDFields() []string      { return cloneStrings(scheduleJobIDFields) }
func MessageIDFields() []string          { return cloneStrings(messageIDFields) }
func MessageTypeFields() []string        { return cloneStrings(messageTypeFields) }
func MessageStatusFields() []string      { return cloneStrings(messageStatusFields) }
func MessageTargetTypeFields() []string  { return cloneStrings(messageTargetTypeFields) }
func MessageTargetIDFields() []string    { return cloneStrings(messageTargetIDFields) }
func MessageContentFields() []string     { return cloneStrings(messageContentFields) }
func ProductDomainIDFields() []string    { return cloneStrings(productDomainIDFields) }
func ProductDomainNameFields() []string  { return cloneStrings(productDomainNameFields) }
func ProductDomainCodeFields() []string  { return cloneStrings(productDomainCodeFields) }
func FAQTitleFields() []string           { return cloneStrings(faqTitleFields) }
func FAQTypeFields() []string            { return cloneStrings(faqTypeFields) }
func LocaleCodeFields() []string         { return cloneStrings(localeCodeFields) }
func LocaleNameFields() []string         { return cloneStrings(localeNameFields) }
func NativeNameFields() []string         { return cloneStrings(nativeNameFields) }
func CodeDescriptionFields() []string    { return cloneStrings(codeDescriptionFields) }
func EnglishDescriptionFields() []string { return cloneStrings(englishDescriptionFields) }
func ThingCategoryIDFields() []string    { return cloneStrings(thingCategoryIDFields) }
func ThingCategoryNameFields() []string  { return cloneStrings(thingCategoryNameFields) }
func ThingCategoryCodeFields() []string  { return cloneStrings(thingCategoryCodeFields) }
func ThingComponentCodeFields() []string { return cloneStrings(thingComponentCodeFields) }
func ThingComponentTypeFields() []string { return cloneStrings(thingComponentTypeFields) }
func ThingPropertyIDFields() []string    { return cloneStrings(thingPropertyIDFields) }
func ThingPropertyNameFields() []string  { return cloneStrings(thingPropertyNameFields) }
func ThingPropertyCodeFields() []string  { return cloneStrings(thingPropertyCodeFields) }
func ThingEventTypeFields() []string     { return cloneStrings(thingEventTypeFields) }

func SupportedBridgeTypeField() string { return internalSupportedBridge }
func ConfigsField() string             { return internalConfigs }
func SubDevicesField() string          { return internalSubDevices }
func FAQItemsField() string            { return internalFAQItems }
func PropsField() string               { return internalProps }
func MeshGroupIDField() string         { return internalMeshGroupID }
func ImportRoomTempIDField() string    { return internalRoomTempID }

func cloneResponseMappings(source []ResponseFieldMapping) []ResponseFieldMapping {
	result := make([]ResponseFieldMapping, 0, len(source))
	for _, item := range source {
		result = append(result, ResponseFieldMapping{Public: item.Public, Internal: cloneStrings(item.Internal)})
	}
	return result
}
