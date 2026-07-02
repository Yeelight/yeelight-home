package semantic

import "strings"

// FieldMapping records the CLI-facing field and the downstream field used by
// the adapter. Callers see only Public; Internal is an implementation detail.
type FieldMapping struct {
	Domain   string
	Public   string
	Internal string
}

type PublicFieldMapping struct {
	Domain string
	Public string
}

const (
	DomainCommon     = "common"
	DomainAction     = "action"
	DomainAutomation = "automation"
	DomainProduct    = "product"
	DomainImport     = "lightingDesignImport"
	DomainPanel      = "panel"
	DomainKnob       = "knob"
	DomainFavorite   = "favorite"
	DomainSort       = "sort"
	DomainHomeMember = "homeMember"
	DomainPreview    = "preview"
	DomainMetrics    = "metrics"
	DomainState      = "state"
)

const (
	FieldKey                              = "key"
	FieldName                             = "name"
	FieldNames                            = "names"
	FieldTitle                            = "title"
	FieldContractVersion                  = "contractVersion"
	FieldRequestID                        = "requestId"
	FieldSessionID                        = "sessionId"
	FieldClientID                         = "clientId"
	FieldLocale                           = "locale"
	FieldUtterance                        = "utterance"
	FieldHomeRef                          = "homeRef"
	FieldConversationContext              = "conversationContext"
	FieldOK                               = "ok"
	FieldQRDevice                         = "qrDevice"
	FieldQRPng                            = "qrPng"
	FieldCredentials                      = "credentials"
	FieldAuthenticated                    = "authenticated"
	FieldAccessTokenPresent               = "accessTokenPresent"
	FieldTokenPresent                     = "tokenPresent"
	FieldTokenSource                      = "tokenSource"
	FieldTokenStore                       = "tokenStore"
	FieldExpireAt                         = "expireAt"
	FieldQRCodeID                         = "qrCodeId"
	FieldActive                           = "active"
	FieldActiveProfile                    = "activeProfile"
	FieldProfiles                         = "profiles"
	FieldDescription                      = "description"
	FieldEnglishDescription               = "enDescription"
	FieldImage                            = "image"
	FieldSequence                         = "sequence"
	FieldIcon                             = "icon"
	FieldHouseID                          = "houseId"
	FieldHouseCount                       = "houseCount"
	FieldHouses                           = "houses"
	FieldAccountOK                        = "accountOk"
	FieldHouseListOK                      = "houseListOk"
	FieldHouseListSource                  = "houseListSource"
	FieldHouseListAPICalls                = "houseListApiCalls"
	FieldID                               = "id"
	FieldIDs                              = "ids"
	FieldEntityID                         = "entityId"
	FieldEntityIDs                        = "entityIds"
	FieldRank                             = "rank"
	FieldIndex                            = "index"
	FieldSubIndex                         = "subIndex"
	FieldRoomID                           = "roomId"
	FieldDeviceID                         = "deviceId"
	FieldDeviceIdentifier                 = "deviceIdentifier"
	FieldDevice                           = "device"
	FieldDeviceName                       = "deviceName"
	FieldGroupID                          = "groupId"
	FieldGroupName                        = "groupName"
	FieldSceneID                          = "sceneId"
	FieldSceneName                        = "sceneName"
	FieldAutomationID                     = "automationId"
	FieldAutomationName                   = "automationName"
	FieldAutomationIDs                    = "automationIds"
	FieldGatewayID                        = "gatewayId"
	FieldPanelName                        = "panelName"
	FieldPanelID                          = "panelId"
	FieldKnobName                         = "knobName"
	FieldKnobID                           = "knobId"
	FieldSensorID                         = "sensorId"
	FieldAreaID                           = "areaId"
	FieldAreaIDs                          = "areaIds"
	FieldArea                             = "area"
	FieldAreaCode                         = "areaCode"
	FieldAreaName                         = "areaName"
	FieldFullName                         = "fullName"
	FieldLevel                            = "level"
	FieldFetchWeather                     = "fetchWeather"
	FieldLeaf                             = "leaf"
	FieldLanguage                         = "language"
	FieldLanguageCode                     = "languageCode"
	FieldDimension                        = "dimension"
	FieldTimeStart                        = "timeStart"
	FieldTimeEnd                          = "timeEnd"
	FieldLatitude                         = "latitude"
	FieldLongitude                        = "longitude"
	FieldFavoriteID                       = "favoriteId"
	FieldFavoriteIDs                      = "favoriteIds"
	FieldFAQID                            = "faqId"
	FieldParentID                         = "parentId"
	FieldVersion                          = "version"
	FieldStatus                           = "status"
	FieldTargetType                       = "targetType"
	FieldTargetTypeID                     = "targetTypeId"
	FieldTargetID                         = "targetId"
	FieldTargetKey                        = "targetKey"
	FieldTargetName                       = "targetName"
	FieldEntityType                       = "entityType"
	FieldEntityName                       = "entityName"
	FieldEntityNames                      = "entityNames"
	FieldCurrentName                      = "currentName"
	FieldNewName                          = "newName"
	FieldAction                           = "action"
	FieldConditionType                    = "conditionType"
	FieldConditions                       = "conditions"
	FieldConditionKind                    = "conditionKind"
	FieldTrigger                          = "trigger"
	FieldTime                             = "time"
	FieldProperty                         = "property"
	FieldOperation                        = "operation"
	FieldOperators                        = "operators"
	FieldValue                            = "value"
	FieldSet                              = "set"
	FieldToggle                           = "toggle"
	FieldAdjust                           = "adjust"
	FieldDelay                            = "delay"
	FieldDuration                         = "duration"
	FieldDelayOff                         = "delayoff"
	FieldFlow                             = "flow"
	FieldCustom                           = "custom"
	FieldPower                            = "power"
	FieldBrightness                       = "brightness"
	FieldColorTemperature                 = "colorTemperature"
	FieldColor                            = "color"
	FieldTargetPercent                    = "targetPercent"
	FieldSwitchPower                      = "switchPower"
	FieldProduct                          = "product"
	FieldProducts                         = "products"
	FieldSKUCode                          = "skuCode"
	FieldProductCode                      = FieldSKUCode
	FieldCapabilityPID                    = "capabilityPid"
	FieldCapabilityPIDs                   = "capabilityPids"
	FieldCapabilityProductID              = FieldCapabilityPID
	FieldCapabilityProductIDs             = FieldCapabilityPIDs
	FieldProductComponentID               = "productComponentId"
	FieldProductCategoryID                = FieldProductComponentID
	FieldProductName                      = "productName"
	FieldProductSKU                       = "productSku"
	FieldProductSPU                       = "productSpu"
	FieldProductBrand                     = "productBrand"
	FieldProductModel                     = "productModel"
	FieldProductLine                      = "productLine"
	FieldProductCategory                  = "productCategoryName"
	FieldProductLargeClass                = "productLargeClass"
	FieldProductSmallClass                = "productSmallClass"
	FieldProductShortName                 = "productShortName"
	FieldProductSeries                    = "productSeries"
	FieldBarcode                          = "barcode"
	FieldBaseUnit                         = "baseUnit"
	FieldProductDeclareNo                 = "productDeclareNo"
	FieldProductDeclareName               = "productDeclareName"
	FieldProductDeclareUnit               = "productDeclareUnit"
	FieldSupportYeelightPro               = "isSupportYeelightPro"
	FieldSupportHomeKit                   = "isSupportHomekit"
	FieldProductStatusName                = "productStatusName"
	FieldExtraMeta                        = "extraMeta"
	FieldModelNo                          = "modelNo"
	FieldPediaDisplay                     = "pediaDisplay"
	FieldProductSaleType                  = "productSaleTypeName"
	FieldQuotationType                    = "quotationTypeDesc"
	FieldProductTypeName                  = "productTypeName"
	FieldCategory                         = "category"
	FieldSeries                           = "series"
	FieldNotes                            = "notes"
	FieldConnectType                      = "connectType"
	FieldComponentName                    = "componentName"
	FieldComponentID                      = "componentId"
	FieldProductEvidence                  = "productEvidence"
	FieldGroupCategory                    = "groupCategory"
	FieldGroupCapability                  = "groupCapability"
	FieldCompatibleSlotKeys               = "compatibleSlotKeys"
	FieldSlotKeys                         = "slotKeys"
	FieldRoomKeys                         = "roomKeys"
	FieldDeviceSlots                      = "deviceSlots"
	FieldDevices                          = "devices"
	FieldGroups                           = "groups"
	FieldAreas                            = "areas"
	FieldScenes                           = "scenes"
	FieldAutomations                      = "automations"
	FieldGateways                         = "gateways"
	FieldAttributes                       = "attributes"
	FieldRules                            = "rules"
	FieldSupported                        = "supported"
	FieldImplemented                      = "implemented"
	FieldSupportedV2                      = "supportedV2"
	FieldSupportedVersions                = "supportedVersions"
	FieldStats                            = "stats"
	FieldThreadInfo                       = "threadInfo"
	FieldSensors                          = "sensors"
	FieldWeather                          = "weather"
	FieldMembers                          = "members"
	FieldFavorites                        = "favorites"
	FieldPanels                           = "panels"
	FieldControls                         = "controls"
	FieldDomains                          = "domains"
	FieldCategories                       = "categories"
	FieldFAQs                             = "faqs"
	FieldFAQ                              = "faq"
	FieldFAQTypes                         = "faqTypes"
	FieldFAQItemTypes                     = "faqItemTypes"
	FieldLocales                          = "locales"
	FieldScheduleJobs                     = "scheduleJobs"
	FieldSchemas                          = "schemas"
	FieldSchema                           = "schema"
	FieldMessages                         = "messages"
	FieldActions                          = "actions"
	FieldDetails                          = "details"
	FieldGateway                          = "gateway"
	FieldGatewayName                      = "gatewayName"
	FieldGatewayDeviceID                  = "gatewayDeviceId"
	FieldGatewayDeviceIDs                 = "gatewayDeviceIds"
	FieldStartTime                        = "startTime"
	FieldEndTime                          = "endTime"
	FieldActiveWindow                     = "activeWindow"
	FieldStart                            = "start"
	FieldEnd                              = "end"
	FieldRepeat                           = "repeat"
	FieldRepeatDays                       = "repeatDays"
	FieldButtonEventID                    = "buttonEventId"
	FieldButtonEvent                      = "buttonEvent"
	FieldButtonEvents                     = "buttonEvents"
	FieldButtonType                       = "buttonType"
	FieldButtons                          = "buttons"
	FieldKeyValue                         = "keyValue"
	FieldAlias                            = "alias"
	FieldVisible                          = "visible"
	FieldEnabled                          = "enabled"
	FieldExtend                           = "extend"
	FieldSort                             = "sort"
	FieldSortType                         = "sortType"
	FieldReadback                         = "readback"
	FieldBackendEvidence                  = "backendEvidence"
	FieldController                       = "controller"
	FieldAdapter                          = "adapter"
	FieldType                             = "type"
	FieldTarget                           = "target"
	FieldOptions                          = "options"
	FieldConfigType                       = "configType"
	FieldMode                             = "mode"
	FieldModel                            = "model"
	FieldModuleID                         = "moduleId"
	FieldSensitivity                      = "sensitivity"
	FieldItems                            = "items"
	FieldSingle                           = "single"
	FieldMulti                            = "multi"
	FieldValid                            = "valid"
	FieldRooms                            = "rooms"
	FieldAddAreaIDs                       = "addAreaIds"
	FieldRemoveAreaIDs                    = "removeAreaIds"
	FieldAddAreaNames                     = "addAreaNames"
	FieldRemoveAreaNames                  = "removeAreaNames"
	FieldRoomIDs                          = "roomIds"
	FieldRoomNames                        = "roomNames"
	FieldDeviceIDs                        = "deviceIds"
	FieldDeviceNames                      = "deviceNames"
	FieldGroupIDs                         = "groupIds"
	FieldSceneIDs                         = "sceneIds"
	FieldGatewayIDs                       = "gatewayIds"
	FieldDefaultGatewayIDs                = "defaultGatewayIds"
	FieldRoomName                         = "roomName"
	FieldTargetRoomID                     = "targetRoomId"
	FieldTargetRoomName                   = "targetRoomName"
	FieldBuildingName                     = "buildingName"
	FieldBuildingAddress                  = "buildingAddr"
	FieldFloorName                        = "floorName"
	FieldMAC                              = "mac"
	FieldCapability                       = "capability"
	FieldDelta                            = "delta"
	FieldStep                             = "step"
	FieldHex                              = "hex"
	FieldRed                              = "red"
	FieldGreen                            = "green"
	FieldBlue                             = "blue"
	FieldKeyword                          = "keyword"
	FieldLimit                            = "limit"
	FieldMemberID                         = "memberId"
	FieldMemberName                       = "memberName"
	FieldMeshGroupID                      = "meshgroupId"
	FieldMultiField                       = "multiField"
	FieldNodeID                           = "nodeId"
	FieldNodeType                         = "nodeType"
	FieldPageNo                           = "pageNo"
	FieldPageSize                         = "pageSize"
	FieldProgressID                       = "progressId"
	FieldProgressKey                      = "progressKey"
	FieldProgress                         = "progress"
	FieldSchemaID                         = "schemaId"
	FieldShareID                          = "shareId"
	FieldSKU                              = "sku"
	FieldSPU                              = "spu"
	FieldUserRole                         = "userRole"
	FieldRole                             = "role"
	FieldUID                              = "uid"
	FieldUserID                           = "userId"
	FieldHomeName                         = "homeName"
	FieldHouseName                        = "houseName"
	FieldUseCurrent                       = "useCurrent"
	FieldPreviewOnly                      = "previewOnly"
	FieldDryRun                           = "dryRun"
	FieldPreview                          = "preview"
	FieldConfirmed                        = "confirmed"
	FieldData                             = "data"
	FieldResult                           = "result"
	FieldReturned                         = "returned"
	FieldAnswer                           = "answer"
	FieldAccount                          = "account"
	FieldCLI                              = "cli"
	FieldPublicRepo                       = "publicRepo"
	FieldOS                               = "os"
	FieldOSType                           = "osType"
	FieldAppType                          = "appType"
	FieldArch                             = "arch"
	FieldExecutable                       = "executable"
	FieldExecutableResolved               = "executableResolved"
	FieldPathLookup                       = "pathLookup"
	FieldPathLookupResolved               = "pathLookupResolved"
	FieldNPMWrapper                       = "npmWrapper"
	FieldNPMWrapperResolved               = "npmWrapperResolved"
	FieldPackageManagers                  = "packageManagers"
	FieldNPM                              = "npm"
	FieldHomebrew                         = "homebrew"
	FieldGitHubRelease                    = "githubRelease"
	FieldHomebrewCask                     = "homebrewCask"
	FieldLatest                           = "latest"
	FieldLatestFile                       = "latestFile"
	FieldAvailable                        = "available"
	FieldInstalled                        = "installed"
	FieldGlobalRoot                       = "globalRoot"
	FieldPackagePath                      = "packagePath"
	FieldPrefix                           = "prefix"
	FieldFormula                          = "formula"
	FieldCask                             = "cask"
	FieldChannel                          = "channel"
	FieldChannels                         = "channels"
	FieldChecked                          = "checked"
	FieldSchemaVersion                    = "schemaVersion"
	FieldCachePolicy                      = "cachePolicy"
	FieldTTLSeconds                       = "ttlSeconds"
	FieldPersistent                       = "persistent"
	FieldTag                              = "tag"
	FieldPublishedAt                      = "publishedAt"
	FieldURL                              = "url"
	FieldCommit                           = "commit"
	FieldDate                             = "date"
	FieldHomeDir                          = "homeDir"
	FieldConfigDir                        = "configDir"
	FieldDataDir                          = "dataDir"
	FieldCacheDir                         = "cacheDir"
	FieldFiles                            = "files"
	FieldInstall                          = "install"
	FieldMemoryMigrations                 = "memoryMigrations"
	FieldRemediations                     = "remediations"
	FieldPrecedence                       = "precedence"
	FieldRootSHA256                       = "rootSha256"
	FieldPersistentWrites                 = "persistentWrites"
	FieldCloudWrites                      = "cloudWrites"
	FieldUnknownEvidence                  = "unknownEvidence"
	FieldEntityEvidence                   = "entityEvidence"
	FieldGuidance                         = "guidance"
	FieldReason                           = "reason"
	FieldBlockReason                      = "blockReason"
	FieldClarification                    = "clarification"
	FieldPlanType                         = "planType"
	FieldRequiredFields                   = "requiredFields"
	FieldPayloadShape                     = "payloadShape"
	FieldExamples                         = "examples"
	FieldNextStep                         = "nextStep"
	FieldNext                             = "next"
	FieldItemsAsMap                       = "itemsAsMap"
	FieldCount                            = "count"
	FieldCreated                          = "created"
	FieldMerged                           = "merged"
	FieldCreatedCount                     = "createdCount"
	FieldMergedCount                      = "mergedCount"
	FieldCreatedAt                        = "createdAt"
	FieldUpdatedAt                        = "updatedAt"
	FieldDeletedCount                     = "deletedCount"
	FieldExport                           = "export"
	FieldNamespace                        = "namespace"
	FieldAccountProfile                   = "accountProfile"
	FieldProfile                          = "profile"
	FieldDataType                         = "dataType"
	FieldLearningEnabled                  = "learningEnabled"
	FieldPaused                           = "paused"
	FieldConsentVersion                   = "consentVersion"
	FieldScopeType                        = "scopeType"
	FieldScopeRef                         = "scopeRef"
	FieldPreferenceID                     = "preferenceId"
	FieldPreferenceIDs                    = "preferenceIds"
	FieldPreferenceType                   = "preferenceType"
	FieldPreferenceValue                  = "preferenceValue"
	FieldPreferences                      = "preferences"
	FieldMemories                         = "memories"
	FieldRecommendation                   = "recommendation"
	FieldRecommendations                  = "recommendations"
	FieldRecommendationID                 = "recommendationId"
	FieldRecommendationIDs                = "recommendationIds"
	FieldRecommendationType               = "recommendationType"
	FieldFeedback                         = "feedback"
	FieldFeedbackRecorded                 = "feedbackRecorded"
	FieldCooldownHours                    = "cooldownHours"
	FieldCooldownUntil                    = "cooldownUntil"
	FieldSessionLimit                     = "sessionLimit"
	FieldItem                             = "item"
	FieldKind                             = "kind"
	FieldEvidence                         = "evidence"
	FieldExplanation                      = "explanation"
	FieldPriority                         = "priority"
	FieldConfidence                       = "confidence"
	FieldActionHint                       = "actionHint"
	FieldActionSource                     = "actionSource"
	FieldParametersHint                   = "parametersHint"
	FieldTargetIntent                     = "targetIntent"
	FieldOperationIntent                  = "operationIntent"
	FieldLesson                           = "lesson"
	FieldOperationLesson                  = "operationLesson"
	FieldOperationLessons                 = "operationLessons"
	FieldLessons                          = "lessons"
	FieldLessonType                       = "lessonType"
	FieldSymptom                          = "symptom"
	FieldCause                            = "cause"
	FieldRecommendedPath                  = "recommendedPath"
	FieldAvoid                            = "avoid"
	FieldFallbackIntent                   = "fallbackIntent"
	FieldStale                            = "stale"
	FieldHitCount                         = "hitCount"
	FieldLastValidatedAt                  = "lastValidatedAt"
	FieldMinConfidence                    = "minConfidence"
	FieldConfidenceAtLeast                = "confidenceAtLeast"
	FieldIncludeStale                     = "includeStale"
	FieldIncludeRejected                  = "includeRejected"
	FieldQuery                            = "query"
	FieldQueryType                        = "queryType"
	FieldFuzzyName                        = "fuzzyName"
	FieldExperience                       = "experience"
	FieldDelegatedIntent                  = "delegatedIntent"
	FieldTemporaryControl                 = "temporaryControl"
	FieldTestOnly                         = "testOnly"
	FieldExpiresAt                        = "expiresAt"
	FieldCreateTime                       = "createTime"
	FieldReuseBarcode                     = "reuseBarcode"
	FieldToUID                            = "toUid"
	FieldTargetMember                     = "targetMember"
	FieldMemberIDMasked                   = "memberIdMasked"
	FieldDisplayName                      = "displayName"
	FieldImpact                           = "impact"
	FieldAffectedScope                    = "affectedScope"
	FieldCallerShouldConfirm              = "callerShouldConfirm"
	FieldRuntimeApprovalStateStored       = "runtimeApprovalStateStored"
	FieldScope                            = "scope"
	FieldRisk                             = "risk"
	FieldIntent                           = "intent"
	FieldLocalOnly                        = "localOnly"
	FieldHouseIndependent                 = "houseIndependent"
	FieldTargetEntityType                 = "targetEntityType"
	FieldTargetIDFlags                    = "targetIdFlags"
	FieldRequestSchema                    = "requestSchema"
	FieldPayloadGuide                     = "payloadGuide"
	FieldExampleCommand                   = "exampleCommand"
	FieldIntentExplanation                = "intentExplanation"
	FieldDirectFields                     = "directFields"
	FieldAllowedIntents                   = "allowedIntents"
	FieldApplyIntent                      = "applyIntent"
	FieldApplyBehavior                    = "applyBehavior"
	FieldSummary                          = "summary"
	FieldExecutionModel                   = "executionModel"
	FieldPreparedForDirectExecution       = "preparedForDirectExecution"
	FieldPayloadPreview                   = "payloadPreview"
	FieldSemanticPreview                  = "semanticPreview"
	FieldPayload                          = "payload"
	FieldParameters                       = "parameters"
	FieldTargets                          = "targets"
	FieldPreconditions                    = "preconditions"
	FieldDestructive                      = "destructive"
	FieldCurrent                          = "current"
	FieldCurrentItems                     = "currentItems"
	FieldCurrentRoomID                    = "currentRoomId"
	FieldPlanned                          = "planned"
	FieldCandidates                       = "candidates"
	FieldSupportedEntityTypes             = "supportedEntityTypes"
	FieldRegion                           = "region"
	FieldSource                           = "source"
	FieldQueryScope                       = "queryScope"
	FieldRawShape                         = "rawShape"
	FieldPropertyName                     = "propertyName"
	FieldProperties                       = "properties"
	FieldSkippedProperties                = "skippedProperties"
	FieldDiagnosticType                   = "diagnosticType"
	FieldStateSource                      = "stateSource"
	FieldStateShape                       = "stateShape"
	FieldExecutionIntent                  = "executionIntent"
	FieldExecutionReadiness               = "executionReadiness"
	FieldExplanationScope                 = "explanationScope"
	FieldDetail                           = "detail"
	FieldEditablePayload                  = "editablePayload"
	FieldUpdateShape                      = "updateShape"
	FieldInputType                        = "inputType"
	FieldCompleteList                     = "completeList"
	FieldRequired                         = "required"
	FieldEditFlow                         = "editFlow"
	FieldCompleteRule                     = "completeRule"
	FieldStatusChange                     = "statusChange"
	FieldAPICalls                         = "apiCalls"
	FieldCacheHits                        = "cacheHits"
	FieldTopologyCacheRefreshCalls        = "topologyCacheRefreshApiCalls"
	FieldTopologyCacheWriteSource         = "topologyCacheWriteSource"
	FieldItemCount                        = "itemCount"
	FieldDeleteTargets                    = "deleteTargets"
	FieldDeleteTarget                     = "deleteTarget"
	FieldMatchedBy                        = "matchedBy"
	FieldFanOut                           = "fanOut"
	FieldBeforeValue                      = "beforeValue"
	FieldExpectedValue                    = "expectedValue"
	FieldVerified                         = "verified"
	FieldVerification                     = "verification"
	FieldVerifiedBy                       = "verifiedBy"
	FieldVerifiedValue                    = "verifiedValue"
	FieldVerifiedTopology                 = "verifiedTopology"
	FieldCommand                          = "command"
	FieldAcceptedFields                   = "acceptedFields"
	FieldAcceptedValueFields              = "acceptedValueFields"
	FieldEntity                           = "entity"
	FieldDesign                           = "design"
	FieldDeviceEvidence                   = "deviceEvidence"
	FieldTraceID                          = "traceId"
	FieldWarnings                         = "warnings"
	FieldUserMessage                      = "userMessage"
	FieldMessage                          = "message"
	FieldError                            = "error"
	FieldStepCount                        = "stepCount"
	FieldSteps                            = "steps"
	FieldCompletedSteps                   = "completedSteps"
	FieldFailedStep                       = "failedStep"
	FieldExclusions                       = "exclusions"
	FieldWritePolicy                      = "writePolicy"
	FieldPreviewUnavailable               = "previewUnavailable"
	FieldWarning                          = "warning"
	FieldPlannedItems                     = "plannedItems"
	FieldProductResolution                = "productResolution"
	FieldCreatesDeviceSlots               = "createsDeviceSlots"
	FieldDeviceSlotsPhysical              = "deviceSlotsArePhysicalBindings"
	FieldTargetMode                       = "targetMode"
	FieldMatchedDeviceSlots               = "matchedDeviceSlots"
	FieldUnresolvedDeviceSlots            = "unresolvedDeviceSlots"
	FieldCatalog                          = "catalog"
	FieldSamples                          = "samples"
	FieldMappings                         = "mappings"
	FieldRequestKey                       = "requestKey"
	FieldSelectedHouseID                  = "selectedHouseId"
	FieldResultData                       = "resultData"
	FieldDeviceCount                      = "deviceCount"
	FieldChildDeviceCount                 = "childDeviceCount"
	FieldConfigs                          = "configs"
	FieldConfigCount                      = "configCount"
	FieldDeviceCountInRoom                = "deviceCountInRoom"
	FieldRoomCount                        = "roomCount"
	FieldGroupCountInRoom                 = "groupCountInRoom"
	FieldRoomCountTotal                   = "roomCountTotal"
	FieldCreatedArtifacts                 = "createdArtifacts"
	FieldActionCount                      = "actionCount"
	FieldResults                          = "results"
	FieldStatusLabel                      = "statusLabel"
	FieldClearMAC                         = "clearMac"
	FieldUnbindRelatedDevices             = "unbindRelDevices"
	FieldRecovery                         = "recovery"
	FieldSuggestedIntent                  = "suggestedIntent"
	FieldSafeNextStep                     = "safeNextStep"
	FieldCanRegenerate                    = "canRegenerate"
	FieldSafeToRetry                      = "safeToRetry"
	FieldTargetDeviceCount                = "targetDeviceCount"
	FieldSupportedProperties              = "supportedProperties"
	FieldSkipped                          = "skipped"
	FieldRuntimeMs                        = "runtimeMs"
	FieldPolicyStatus                     = "policyStatus"
	FieldHTTPStatus                       = "httpStatus"
	FieldNextAction                       = "nextAction"
	FieldCapabilitySource                 = "capabilitySource"
	FieldSchemaStatus                     = "schemaStatus"
	FieldDeviceSchema                     = "deviceSchema"
	FieldOperations                       = "operations"
	FieldLimitations                      = "limitations"
	FieldRead                             = "read"
	FieldWrite                            = "write"
	FieldOnline                           = "online"
	FieldBind                             = "bind"
	FieldVirtual                          = "virtual"
	FieldPartial                          = "partial"
	FieldPartialState                     = "partialState"
	FieldMacMasked                        = "macMasked"
	FieldPhoneMasked                      = "phoneMasked"
	FieldEmailMasked                      = "emailMasked"
	FieldSupportedBridgeType              = "supportedBridgeType"
	FieldFirmwareVersion                  = "firmwareVersion"
	FieldFirmwareType                     = "firmwareType"
	FieldCurrentVersion                   = "currentVersion"
	FieldTypeName                         = "typeName"
	FieldRoomRank                         = "roomRank"
	FieldRuleID                           = "ruleId"
	FieldTimeInterval                     = "timeInterval"
	FieldVirtualDeviceCount               = "virtualDeviceCount"
	FieldUnboundDevices                   = "unboundDevices"
	FieldUnboundGateways                  = "unboundGateways"
	FieldFeatures                         = "features"
	FieldValueRange                       = "valueRange"
	FieldSupportActions                   = "supportActions"
	FieldQueryList                        = "queryList"
	FieldFAQItems                         = "faqItems"
	FieldPosition                         = "position"
	FieldOrder                            = "order"
	FieldOrderBy                          = "orderBy"
	FieldTotal                            = "total"
	FieldCounts                           = "counts"
	FieldExpectedCounts                   = "expectedCounts"
	FieldObservedCounts                   = "observedCounts"
	FieldEntries                          = "entries"
	FieldScript                           = "script"
	FieldResourceStatus                   = "resourceStatus"
	FieldResourceIndex                    = "resIndex"
	FieldResourceTypeID                   = "resourceTypeId"
	FieldAttachments                      = "attachments"
	FieldResources                        = "resources"
	FieldManualCandidateURL               = "manualCandidateUrl"
	FieldFAQCandidateURL                  = "faqCandidateUrl"
	FieldCandidateStatus                  = "candidateStatus"
	FieldManualAttachments                = "manualAttachments"
	FieldEncryption                       = "encryption"
	FieldImportPolicy                     = "importPolicy"
	FieldRetentionPolicy                  = "retentionPolicy"
	FieldConsents                         = "consents"
	FieldSignals                          = "signals"
	FieldSignalType                       = "signalType"
	FieldSignalKey                        = "signalKey"
	FieldFirstSeenAt                      = "firstSeenAt"
	FieldLastSeenAt                       = "lastSeenAt"
	FieldLastShownAt                      = "lastShownAt"
	FieldExplicitPreferences              = "explicitPreferences"
	FieldRecommendationEvidenceDays       = "recommendationEvidenceDays"
	FieldRecommendationCompactionScope    = "recommendationCompactionScope"
	FieldPendingRecommendations           = "pendingRecommendations"
	FieldInteractionEventsDays            = "interactionEventsDays"
	FieldInteractionEvidence              = "interactionEvidence"
	FieldOperationLessonsRetention        = "operationLessons"
	FieldRuntimeSubjectiveInferencePolicy = "runtimeSubjectiveInferencePolicy"
	FieldEntities                         = "entities"
	FieldComponents                       = "components"
	FieldEvents                           = "events"
	FieldInputs                           = "inputs"
	FieldAccess                           = "access"
	FieldFormat                           = "format"
	FieldUnit                             = "unit"
	FieldRange                            = "range"
	FieldMin                              = "min"
	FieldMax                              = "max"
	FieldValueList                        = "valueList"
	FieldCode                             = "code"
	FieldEventID                          = "eventId"
	FieldEventTypeID                      = "eventTypeId"
	FieldEventNo                          = "eventNo"
	FieldEventUnitNum                     = "eventUnitNum"
	FieldEventArgs                        = "eventArgs"
	FieldNativeName                       = "nativeName"
	FieldPropertyCount                    = "propertyCount"
	FieldComponentCount                   = "componentCount"
	FieldEventCount                       = "eventCount"
	FieldReadable                         = "readable"
	FieldWritable                         = "writable"
)

const (
	internalTypeID            = "typeId"
	internalResourceID        = "resId"
	internalResourceName      = "resName"
	internalResourceType      = "resType"
	internalParams            = "params"
	internalParam             = "param"
	internalSubIndex          = "idx"
	internalTempID            = "tempId"
	internalRoomList          = "roomList"
	internalDeviceList        = "deviceList"
	internalGroupList         = "groupList"
	internalAreaList          = "areaList"
	internalSceneList         = "sceneList"
	internalAutomationList    = "automationList"
	internalRoomTempID        = "roomTempId"
	internalRoomTempIDList    = "roomTempIdList"
	internalDeviceTempIDList  = "deviceTempIdList"
	internalComponentID       = "componentId"
	internalProductCode       = "materialCode"
	internalProductID         = "pid"
	internalProductIDs        = "pids"
	internalProductCategoryID = "pcId"
	internalCloudComponentID  = "cid"
	internalDeviceIdentifier  = "did"
	internalPower             = "p"
	internalBrightness        = "l"
	internalColorTemperature  = "ct"
	internalColor             = "c"
	internalTargetPercent     = "tp"
	internalSwitchPower       = "sp"
	internalConditionType     = "type"
	internalConditionKind     = "type"
	internalClock             = "clock"
	internalProperty          = "prop"
	internalEventArgs         = "extArgs"
	internalDescription       = "desc"
	internalImage             = "img"
	internalSequence          = "seq"
	internalBindFlag          = "isBind"
	internalVirtualFlag       = "isVirtual"
	internalRepeatType        = "repeatType"
	internalRepeatValue       = "repeatValue"
	internalExpiresAt         = "expiredTime"
	internalSensitivity       = "sens"
	internalMetaImportState   = "state"
	internalUpperHouseID      = "houseID"
	internalAddAreaList       = "addAreaList"
	internalRemoveAreaList    = "removeAreaList"
)

var (
	internalAccountIDCandidateFields          = []string{FieldID, FieldUID, FieldUserID, "accountId"}
	internalAccountDisplayNameCandidateFields = []string{"nickname", "nickName", FieldName, FieldDisplayName}
)

var fieldMappings = []FieldMapping{
	{Domain: DomainCommon, Public: FieldHouseID, Internal: FieldHouseID},
	{Domain: DomainCommon, Public: FieldHouseCount, Internal: FieldHouseCount},
	{Domain: DomainCommon, Public: FieldHouses, Internal: FieldHouses},
	{Domain: DomainCommon, Public: FieldAccountOK, Internal: FieldAccountOK},
	{Domain: DomainCommon, Public: FieldHouseListOK, Internal: FieldHouseListOK},
	{Domain: DomainCommon, Public: FieldHouseListSource, Internal: FieldHouseListSource},
	{Domain: DomainCommon, Public: FieldHouseListAPICalls, Internal: FieldHouseListAPICalls},
	{Domain: DomainCommon, Public: FieldID, Internal: FieldID},
	{Domain: DomainCommon, Public: FieldIDs, Internal: FieldIDs},
	{Domain: DomainCommon, Public: FieldEntityID, Internal: FieldEntityID},
	{Domain: DomainCommon, Public: FieldEntityIDs, Internal: FieldEntityIDs},
	{Domain: DomainCommon, Public: FieldEntityName, Internal: FieldEntityName},
	{Domain: DomainCommon, Public: FieldEntityNames, Internal: FieldEntityNames},
	{Domain: DomainCommon, Public: FieldKey, Internal: FieldKey},
	{Domain: DomainCommon, Public: FieldName, Internal: FieldName},
	{Domain: DomainCommon, Public: FieldNames, Internal: FieldNames},
	{Domain: DomainCommon, Public: FieldTitle, Internal: FieldTitle},
	{Domain: DomainCommon, Public: FieldContractVersion, Internal: FieldContractVersion},
	{Domain: DomainCommon, Public: FieldRequestID, Internal: FieldRequestID},
	{Domain: DomainCommon, Public: FieldSessionID, Internal: FieldSessionID},
	{Domain: DomainCommon, Public: FieldClientID, Internal: FieldClientID},
	{Domain: DomainCommon, Public: FieldLocale, Internal: FieldLocale},
	{Domain: DomainCommon, Public: FieldUtterance, Internal: FieldUtterance},
	{Domain: DomainCommon, Public: FieldHomeRef, Internal: FieldHomeRef},
	{Domain: DomainCommon, Public: FieldConversationContext, Internal: FieldConversationContext},
	{Domain: DomainCommon, Public: FieldOK, Internal: FieldOK},
	{Domain: DomainCommon, Public: FieldQRDevice, Internal: FieldQRDevice},
	{Domain: DomainCommon, Public: FieldQRPng, Internal: FieldQRPng},
	{Domain: DomainCommon, Public: FieldCredentials, Internal: FieldCredentials},
	{Domain: DomainCommon, Public: FieldAuthenticated, Internal: FieldAuthenticated},
	{Domain: DomainCommon, Public: FieldAccessTokenPresent, Internal: FieldAccessTokenPresent},
	{Domain: DomainCommon, Public: FieldTokenPresent, Internal: FieldTokenPresent},
	{Domain: DomainCommon, Public: FieldTokenSource, Internal: FieldTokenSource},
	{Domain: DomainCommon, Public: FieldTokenStore, Internal: FieldTokenStore},
	{Domain: DomainCommon, Public: FieldExpireAt, Internal: FieldExpireAt},
	{Domain: DomainCommon, Public: FieldQRCodeID, Internal: FieldQRCodeID},
	{Domain: DomainCommon, Public: FieldActive, Internal: FieldActive},
	{Domain: DomainCommon, Public: FieldActiveProfile, Internal: FieldActiveProfile},
	{Domain: DomainCommon, Public: FieldProfiles, Internal: FieldProfiles},
	{Domain: DomainCommon, Public: FieldDescription, Internal: internalDescription},
	{Domain: DomainCommon, Public: FieldEnglishDescription, Internal: FieldEnglishDescription},
	{Domain: DomainCommon, Public: FieldImage, Internal: internalImage},
	{Domain: DomainCommon, Public: FieldSequence, Internal: internalSequence},
	{Domain: DomainCommon, Public: FieldIcon, Internal: FieldIcon},
	{Domain: DomainCommon, Public: FieldRank, Internal: FieldRank},
	{Domain: DomainCommon, Public: FieldRoomID, Internal: FieldRoomID},
	{Domain: DomainCommon, Public: FieldDeviceID, Internal: FieldDeviceID},
	{Domain: DomainCommon, Public: FieldDeviceIdentifier, Internal: FieldDeviceIdentifier},
	{Domain: DomainCommon, Public: FieldDevice, Internal: FieldDevice},
	{Domain: DomainCommon, Public: FieldDeviceName, Internal: FieldDeviceName},
	{Domain: DomainCommon, Public: FieldGroupID, Internal: FieldGroupID},
	{Domain: DomainCommon, Public: FieldGroupName, Internal: FieldGroupName},
	{Domain: DomainCommon, Public: FieldSceneID, Internal: FieldSceneID},
	{Domain: DomainCommon, Public: FieldSceneName, Internal: FieldSceneName},
	{Domain: DomainCommon, Public: FieldSceneIDs, Internal: FieldSceneIDs},
	{Domain: DomainCommon, Public: FieldAutomationID, Internal: FieldAutomationID},
	{Domain: DomainCommon, Public: FieldAutomationName, Internal: FieldAutomationName},
	{Domain: DomainCommon, Public: FieldAutomationIDs, Internal: FieldAutomationIDs},
	{Domain: DomainCommon, Public: FieldGatewayID, Internal: FieldGatewayID},
	{Domain: DomainCommon, Public: FieldGatewayIDs, Internal: FieldGatewayIDs},
	{Domain: DomainCommon, Public: FieldPanelName, Internal: FieldPanelName},
	{Domain: DomainCommon, Public: FieldPanelID, Internal: FieldPanelID},
	{Domain: DomainCommon, Public: FieldKnobName, Internal: FieldKnobName},
	{Domain: DomainCommon, Public: FieldKnobID, Internal: FieldKnobID},
	{Domain: DomainCommon, Public: FieldSensorID, Internal: FieldSensorID},
	{Domain: DomainCommon, Public: FieldAreaID, Internal: FieldAreaID},
	{Domain: DomainCommon, Public: FieldAreaIDs, Internal: FieldAreaIDs},
	{Domain: DomainCommon, Public: FieldArea, Internal: FieldArea},
	{Domain: DomainCommon, Public: FieldAreaCode, Internal: FieldAreaCode},
	{Domain: DomainCommon, Public: FieldAreaName, Internal: FieldAreaName},
	{Domain: DomainCommon, Public: FieldFullName, Internal: FieldFullName},
	{Domain: DomainCommon, Public: FieldLevel, Internal: FieldLevel},
	{Domain: DomainCommon, Public: FieldFetchWeather, Internal: FieldFetchWeather},
	{Domain: DomainCommon, Public: FieldLeaf, Internal: FieldLeaf},
	{Domain: DomainCommon, Public: FieldLanguage, Internal: FieldLanguage},
	{Domain: DomainCommon, Public: FieldLanguageCode, Internal: FieldLanguageCode},
	{Domain: DomainCommon, Public: FieldDimension, Internal: FieldDimension},
	{Domain: DomainCommon, Public: FieldTimeStart, Internal: FieldTimeStart},
	{Domain: DomainCommon, Public: FieldTimeEnd, Internal: FieldTimeEnd},
	{Domain: DomainCommon, Public: FieldLatitude, Internal: FieldLatitude},
	{Domain: DomainCommon, Public: FieldLongitude, Internal: FieldLongitude},
	{Domain: DomainCommon, Public: FieldFavoriteID, Internal: FieldFavoriteID},
	{Domain: DomainCommon, Public: FieldFavoriteIDs, Internal: FieldFavoriteIDs},
	{Domain: DomainCommon, Public: FieldFAQID, Internal: FieldFAQID},
	{Domain: DomainCommon, Public: FieldParentID, Internal: FieldParentID},
	{Domain: DomainCommon, Public: FieldVersion, Internal: FieldVersion},
	{Domain: DomainCommon, Public: FieldStatus, Internal: FieldStatus},
	{Domain: DomainCommon, Public: FieldTargetType, Internal: FieldTargetType},
	{Domain: DomainCommon, Public: FieldTargetTypeID, Internal: FieldTargetTypeID},
	{Domain: DomainCommon, Public: FieldTargetID, Internal: FieldTargetID},
	{Domain: DomainCommon, Public: FieldTargetKey, Internal: FieldTargetKey},
	{Domain: DomainCommon, Public: FieldTargetName, Internal: FieldTargetName},
	{Domain: DomainCommon, Public: FieldEntityType, Internal: FieldEntityType},
	{Domain: DomainCommon, Public: FieldCurrentName, Internal: FieldCurrentName},
	{Domain: DomainCommon, Public: FieldNewName, Internal: FieldNewName},
	{Domain: DomainCommon, Public: FieldAction, Internal: FieldAction},
	{Domain: DomainCommon, Public: FieldConditionType, Internal: FieldConditionType},
	{Domain: DomainCommon, Public: FieldConditions, Internal: FieldConditions},
	{Domain: DomainCommon, Public: FieldConditionKind, Internal: FieldConditionKind},
	{Domain: DomainCommon, Public: FieldTrigger, Internal: FieldTrigger},
	{Domain: DomainCommon, Public: FieldTime, Internal: FieldTime},
	{Domain: DomainCommon, Public: FieldProperty, Internal: FieldProperty},
	{Domain: DomainCommon, Public: FieldOperation, Internal: FieldOperation},
	{Domain: DomainCommon, Public: FieldOperators, Internal: FieldOperators},
	{Domain: DomainCommon, Public: FieldValue, Internal: FieldValue},
	{Domain: DomainCommon, Public: FieldSet, Internal: FieldSet},
	{Domain: DomainCommon, Public: FieldToggle, Internal: FieldToggle},
	{Domain: DomainCommon, Public: FieldAdjust, Internal: FieldAdjust},
	{Domain: DomainCommon, Public: FieldDelay, Internal: FieldDelay},
	{Domain: DomainCommon, Public: FieldDuration, Internal: FieldDuration},
	{Domain: DomainCommon, Public: FieldDelayOff, Internal: FieldDelayOff},
	{Domain: DomainCommon, Public: FieldFlow, Internal: FieldFlow},
	{Domain: DomainCommon, Public: FieldCustom, Internal: FieldCustom},
	{Domain: DomainCommon, Public: FieldPower, Internal: FieldPower},
	{Domain: DomainCommon, Public: FieldBrightness, Internal: FieldBrightness},
	{Domain: DomainCommon, Public: FieldColorTemperature, Internal: FieldColorTemperature},
	{Domain: DomainCommon, Public: FieldColor, Internal: FieldColor},
	{Domain: DomainCommon, Public: FieldTargetPercent, Internal: FieldTargetPercent},
	{Domain: DomainCommon, Public: FieldSwitchPower, Internal: FieldSwitchPower},
	{Domain: DomainCommon, Public: FieldItems, Internal: FieldItems},
	{Domain: DomainCommon, Public: FieldSingle, Internal: FieldSingle},
	{Domain: DomainCommon, Public: FieldMulti, Internal: FieldMulti},
	{Domain: DomainCommon, Public: FieldValid, Internal: FieldValid},
	{Domain: DomainCommon, Public: FieldOptions, Internal: FieldOptions},
	{Domain: DomainCommon, Public: FieldTarget, Internal: FieldTarget},
	{Domain: DomainCommon, Public: FieldSortType, Internal: FieldSortType},
	{Domain: DomainCommon, Public: FieldSort, Internal: FieldSort},
	{Domain: DomainCommon, Public: FieldReadback, Internal: FieldReadback},
	{Domain: DomainCommon, Public: FieldBackendEvidence, Internal: FieldBackendEvidence},
	{Domain: DomainCommon, Public: FieldController, Internal: FieldController},
	{Domain: DomainCommon, Public: FieldAdapter, Internal: FieldAdapter},
	{Domain: DomainCommon, Public: FieldEnabled, Internal: FieldEnabled},
	{Domain: DomainCommon, Public: FieldType, Internal: FieldType},
	{Domain: DomainCommon, Public: FieldModel, Internal: FieldModel},
	{Domain: DomainCommon, Public: FieldModuleID, Internal: FieldModuleID},
	{Domain: DomainCommon, Public: FieldRooms, Internal: FieldRooms},
	{Domain: DomainCommon, Public: FieldDevices, Internal: FieldDevices},
	{Domain: DomainCommon, Public: FieldProducts, Internal: FieldProducts},
	{Domain: DomainCommon, Public: FieldGateways, Internal: FieldGateways},
	{Domain: DomainCommon, Public: FieldAttributes, Internal: FieldAttributes},
	{Domain: DomainCommon, Public: FieldRules, Internal: FieldRules},
	{Domain: DomainCommon, Public: FieldSupported, Internal: FieldSupported},
	{Domain: DomainCommon, Public: FieldImplemented, Internal: FieldImplemented},
	{Domain: DomainCommon, Public: FieldSupportedV2, Internal: FieldSupportedV2},
	{Domain: DomainCommon, Public: FieldSupportedVersions, Internal: FieldSupportedVersions},
	{Domain: DomainCommon, Public: FieldStats, Internal: FieldStats},
	{Domain: DomainCommon, Public: FieldThreadInfo, Internal: FieldThreadInfo},
	{Domain: DomainCommon, Public: FieldSensors, Internal: FieldSensors},
	{Domain: DomainCommon, Public: FieldWeather, Internal: FieldWeather},
	{Domain: DomainCommon, Public: FieldMembers, Internal: FieldMembers},
	{Domain: DomainCommon, Public: FieldFavorites, Internal: FieldFavorites},
	{Domain: DomainCommon, Public: FieldPanels, Internal: FieldPanels},
	{Domain: DomainCommon, Public: FieldControls, Internal: FieldControls},
	{Domain: DomainCommon, Public: FieldDomains, Internal: FieldDomains},
	{Domain: DomainCommon, Public: FieldCategories, Internal: FieldCategories},
	{Domain: DomainCommon, Public: FieldFAQs, Internal: FieldFAQs},
	{Domain: DomainCommon, Public: FieldFAQ, Internal: FieldFAQ},
	{Domain: DomainCommon, Public: FieldFAQTypes, Internal: FieldFAQTypes},
	{Domain: DomainCommon, Public: FieldFAQItemTypes, Internal: FieldFAQItemTypes},
	{Domain: DomainCommon, Public: FieldLocales, Internal: FieldLocales},
	{Domain: DomainCommon, Public: FieldScheduleJobs, Internal: FieldScheduleJobs},
	{Domain: DomainCommon, Public: FieldSchemas, Internal: FieldSchemas},
	{Domain: DomainCommon, Public: FieldSchema, Internal: FieldSchema},
	{Domain: DomainCommon, Public: FieldMessages, Internal: FieldMessages},
	{Domain: DomainCommon, Public: FieldRoomIDs, Internal: FieldRoomIDs},
	{Domain: DomainCommon, Public: FieldRoomNames, Internal: FieldRoomNames},
	{Domain: DomainCommon, Public: FieldAddAreaIDs, Internal: internalAddAreaList},
	{Domain: DomainCommon, Public: FieldRemoveAreaIDs, Internal: internalRemoveAreaList},
	{Domain: DomainCommon, Public: FieldAddAreaNames, Internal: FieldAddAreaNames},
	{Domain: DomainCommon, Public: FieldRemoveAreaNames, Internal: FieldRemoveAreaNames},
	{Domain: DomainCommon, Public: FieldRoomName, Internal: FieldRoomName},
	{Domain: DomainCommon, Public: FieldTargetRoomID, Internal: FieldTargetRoomID},
	{Domain: DomainCommon, Public: FieldTargetRoomName, Internal: FieldTargetRoomName},
	{Domain: DomainCommon, Public: FieldDeviceIDs, Internal: FieldDeviceIDs},
	{Domain: DomainCommon, Public: FieldDeviceNames, Internal: FieldDeviceNames},
	{Domain: DomainCommon, Public: FieldGroupIDs, Internal: FieldGroupIDs},
	{Domain: DomainCommon, Public: FieldBuildingName, Internal: FieldBuildingName},
	{Domain: DomainCommon, Public: FieldBuildingAddress, Internal: FieldBuildingAddress},
	{Domain: DomainCommon, Public: FieldFloorName, Internal: FieldFloorName},
	{Domain: DomainCommon, Public: FieldGatewayDeviceID, Internal: FieldGatewayDeviceID},
	{Domain: DomainCommon, Public: FieldGatewayDeviceIDs, Internal: FieldGatewayDeviceIDs},
	{Domain: DomainCommon, Public: FieldComponentID, Internal: FieldComponentID},
	{Domain: DomainCommon, Public: FieldComponentName, Internal: FieldComponentName},
	{Domain: DomainCommon, Public: FieldDefaultGatewayIDs, Internal: FieldDefaultGatewayIDs},
	{Domain: DomainCommon, Public: FieldStartTime, Internal: FieldStartTime},
	{Domain: DomainCommon, Public: FieldEndTime, Internal: FieldEndTime},
	{Domain: DomainCommon, Public: FieldActiveWindow, Internal: FieldActiveWindow},
	{Domain: DomainCommon, Public: FieldStart, Internal: FieldStart},
	{Domain: DomainCommon, Public: FieldEnd, Internal: FieldEnd},
	{Domain: DomainCommon, Public: FieldRepeat, Internal: FieldRepeat},
	{Domain: DomainCommon, Public: FieldRepeatDays, Internal: FieldRepeatDays},
	{Domain: DomainCommon, Public: FieldMAC, Internal: FieldMAC},
	{Domain: DomainCommon, Public: FieldCapability, Internal: FieldCapability},
	{Domain: DomainCommon, Public: FieldDelta, Internal: FieldDelta},
	{Domain: DomainCommon, Public: FieldStep, Internal: FieldStep},
	{Domain: DomainCommon, Public: FieldHex, Internal: FieldHex},
	{Domain: DomainCommon, Public: FieldRed, Internal: FieldRed},
	{Domain: DomainCommon, Public: FieldGreen, Internal: FieldGreen},
	{Domain: DomainCommon, Public: FieldBlue, Internal: FieldBlue},
	{Domain: DomainCommon, Public: FieldKeyword, Internal: FieldKeyword},
	{Domain: DomainCommon, Public: FieldLimit, Internal: FieldLimit},
	{Domain: DomainCommon, Public: FieldMemberID, Internal: FieldMemberID},
	{Domain: DomainCommon, Public: FieldMemberName, Internal: FieldMemberName},
	{Domain: DomainCommon, Public: FieldMeshGroupID, Internal: FieldMeshGroupID},
	{Domain: DomainCommon, Public: FieldMultiField, Internal: FieldMultiField},
	{Domain: DomainCommon, Public: FieldNodeID, Internal: FieldNodeID},
	{Domain: DomainCommon, Public: FieldNodeType, Internal: FieldNodeType},
	{Domain: DomainCommon, Public: FieldPageNo, Internal: FieldPageNo},
	{Domain: DomainCommon, Public: FieldPageSize, Internal: FieldPageSize},
	{Domain: DomainCommon, Public: FieldProgressID, Internal: FieldProgressID},
	{Domain: DomainCommon, Public: FieldProgressKey, Internal: FieldProgressKey},
	{Domain: DomainCommon, Public: FieldProgress, Internal: FieldProgress},
	{Domain: DomainCommon, Public: FieldSchemaID, Internal: FieldSchemaID},
	{Domain: DomainCommon, Public: FieldShareID, Internal: FieldShareID},
	{Domain: DomainCommon, Public: FieldSKU, Internal: FieldSKU},
	{Domain: DomainCommon, Public: FieldSPU, Internal: FieldSPU},
	{Domain: DomainCommon, Public: FieldUserRole, Internal: FieldUserRole},
	{Domain: DomainCommon, Public: FieldRole, Internal: FieldRole},
	{Domain: DomainCommon, Public: FieldUID, Internal: FieldUID},
	{Domain: DomainCommon, Public: FieldUserID, Internal: FieldUserID},
	{Domain: DomainCommon, Public: FieldHomeName, Internal: FieldHomeName},
	{Domain: DomainCommon, Public: FieldHouseName, Internal: FieldHouseName},
	{Domain: DomainCommon, Public: FieldUseCurrent, Internal: FieldUseCurrent},
	{Domain: DomainCommon, Public: FieldPreviewOnly, Internal: FieldPreviewOnly},
	{Domain: DomainCommon, Public: FieldDryRun, Internal: FieldDryRun},
	{Domain: DomainCommon, Public: FieldPreview, Internal: FieldPreview},
	{Domain: DomainCommon, Public: FieldConfirmed, Internal: FieldConfirmed},
	{Domain: DomainCommon, Public: FieldData, Internal: FieldData},
	{Domain: DomainCommon, Public: FieldResult, Internal: FieldResult},
	{Domain: DomainCommon, Public: FieldReturned, Internal: FieldReturned},
	{Domain: DomainCommon, Public: FieldAnswer, Internal: FieldAnswer},
	{Domain: DomainCommon, Public: FieldAccount, Internal: FieldAccount},
	{Domain: DomainCommon, Public: FieldCLI, Internal: FieldCLI},
	{Domain: DomainCommon, Public: FieldPublicRepo, Internal: FieldPublicRepo},
	{Domain: DomainCommon, Public: FieldOS, Internal: FieldOS},
	{Domain: DomainCommon, Public: FieldOSType, Internal: FieldOSType},
	{Domain: DomainCommon, Public: FieldAppType, Internal: FieldAppType},
	{Domain: DomainCommon, Public: FieldArch, Internal: FieldArch},
	{Domain: DomainCommon, Public: FieldExecutable, Internal: FieldExecutable},
	{Domain: DomainCommon, Public: FieldExecutableResolved, Internal: FieldExecutableResolved},
	{Domain: DomainCommon, Public: FieldPathLookup, Internal: FieldPathLookup},
	{Domain: DomainCommon, Public: FieldPathLookupResolved, Internal: FieldPathLookupResolved},
	{Domain: DomainCommon, Public: FieldNPMWrapper, Internal: FieldNPMWrapper},
	{Domain: DomainCommon, Public: FieldNPMWrapperResolved, Internal: FieldNPMWrapperResolved},
	{Domain: DomainCommon, Public: FieldPackageManagers, Internal: FieldPackageManagers},
	{Domain: DomainCommon, Public: FieldNPM, Internal: FieldNPM},
	{Domain: DomainCommon, Public: FieldHomebrew, Internal: FieldHomebrew},
	{Domain: DomainCommon, Public: FieldGitHubRelease, Internal: FieldGitHubRelease},
	{Domain: DomainCommon, Public: FieldHomebrewCask, Internal: FieldHomebrewCask},
	{Domain: DomainCommon, Public: FieldLatest, Internal: FieldLatest},
	{Domain: DomainCommon, Public: FieldLatestFile, Internal: FieldLatestFile},
	{Domain: DomainCommon, Public: FieldAvailable, Internal: FieldAvailable},
	{Domain: DomainCommon, Public: FieldInstalled, Internal: FieldInstalled},
	{Domain: DomainCommon, Public: FieldGlobalRoot, Internal: FieldGlobalRoot},
	{Domain: DomainCommon, Public: FieldPackagePath, Internal: FieldPackagePath},
	{Domain: DomainCommon, Public: FieldPrefix, Internal: FieldPrefix},
	{Domain: DomainCommon, Public: FieldFormula, Internal: FieldFormula},
	{Domain: DomainCommon, Public: FieldCask, Internal: FieldCask},
	{Domain: DomainCommon, Public: FieldChannel, Internal: FieldChannel},
	{Domain: DomainCommon, Public: FieldChannels, Internal: FieldChannels},
	{Domain: DomainCommon, Public: FieldChecked, Internal: FieldChecked},
	{Domain: DomainCommon, Public: FieldSchemaVersion, Internal: FieldSchemaVersion},
	{Domain: DomainCommon, Public: FieldCachePolicy, Internal: FieldCachePolicy},
	{Domain: DomainCommon, Public: FieldTTLSeconds, Internal: FieldTTLSeconds},
	{Domain: DomainCommon, Public: FieldPersistent, Internal: FieldPersistent},
	{Domain: DomainCommon, Public: FieldTag, Internal: FieldTag},
	{Domain: DomainCommon, Public: FieldPublishedAt, Internal: FieldPublishedAt},
	{Domain: DomainCommon, Public: FieldURL, Internal: FieldURL},
	{Domain: DomainCommon, Public: FieldCommit, Internal: FieldCommit},
	{Domain: DomainCommon, Public: FieldDate, Internal: FieldDate},
	{Domain: DomainCommon, Public: FieldHomeDir, Internal: FieldHomeDir},
	{Domain: DomainCommon, Public: FieldConfigDir, Internal: FieldConfigDir},
	{Domain: DomainCommon, Public: FieldDataDir, Internal: FieldDataDir},
	{Domain: DomainCommon, Public: FieldCacheDir, Internal: FieldCacheDir},
	{Domain: DomainCommon, Public: FieldFiles, Internal: FieldFiles},
	{Domain: DomainCommon, Public: FieldInstall, Internal: FieldInstall},
	{Domain: DomainCommon, Public: FieldMemoryMigrations, Internal: FieldMemoryMigrations},
	{Domain: DomainCommon, Public: FieldRemediations, Internal: FieldRemediations},
	{Domain: DomainCommon, Public: FieldPrecedence, Internal: FieldPrecedence},
	{Domain: DomainCommon, Public: FieldRootSHA256, Internal: FieldRootSHA256},
	{Domain: DomainCommon, Public: FieldRawShape, Internal: FieldRawShape},
	{Domain: DomainCommon, Public: FieldDetail, Internal: FieldDetail},
	{Domain: DomainCommon, Public: FieldEditablePayload, Internal: FieldEditablePayload},
	{Domain: DomainCommon, Public: FieldUpdateShape, Internal: FieldUpdateShape},
	{Domain: DomainCommon, Public: FieldInputType, Internal: FieldInputType},
	{Domain: DomainCommon, Public: FieldCompleteList, Internal: FieldCompleteList},
	{Domain: DomainCommon, Public: FieldRequired, Internal: FieldRequired},
	{Domain: DomainCommon, Public: FieldEditFlow, Internal: FieldEditFlow},
	{Domain: DomainCommon, Public: FieldCompleteRule, Internal: FieldCompleteRule},
	{Domain: DomainCommon, Public: FieldStatusChange, Internal: FieldStatusChange},
	{Domain: DomainCommon, Public: FieldPersistentWrites, Internal: FieldPersistentWrites},
	{Domain: DomainCommon, Public: FieldCloudWrites, Internal: FieldCloudWrites},
	{Domain: DomainCommon, Public: FieldUnknownEvidence, Internal: FieldUnknownEvidence},
	{Domain: DomainCommon, Public: FieldEntityEvidence, Internal: FieldEntityEvidence},
	{Domain: DomainCommon, Public: FieldGuidance, Internal: FieldGuidance},
	{Domain: DomainCommon, Public: FieldReason, Internal: FieldReason},
	{Domain: DomainCommon, Public: FieldBlockReason, Internal: FieldBlockReason},
	{Domain: DomainCommon, Public: FieldClarification, Internal: FieldClarification},
	{Domain: DomainCommon, Public: FieldPlanType, Internal: FieldPlanType},
	{Domain: DomainCommon, Public: FieldRequiredFields, Internal: FieldRequiredFields},
	{Domain: DomainCommon, Public: FieldPayloadShape, Internal: FieldPayloadShape},
	{Domain: DomainCommon, Public: FieldExamples, Internal: FieldExamples},
	{Domain: DomainCommon, Public: FieldNextStep, Internal: FieldNextStep},
	{Domain: DomainCommon, Public: FieldNext, Internal: FieldNext},
	{Domain: DomainCommon, Public: FieldItemsAsMap, Internal: FieldItemsAsMap},
	{Domain: DomainCommon, Public: FieldCount, Internal: FieldCount},
	{Domain: DomainCommon, Public: FieldCreated, Internal: FieldCreated},
	{Domain: DomainCommon, Public: FieldMerged, Internal: FieldMerged},
	{Domain: DomainCommon, Public: FieldCreatedCount, Internal: FieldCreatedCount},
	{Domain: DomainCommon, Public: FieldMergedCount, Internal: FieldMergedCount},
	{Domain: DomainCommon, Public: FieldCreatedAt, Internal: FieldCreatedAt},
	{Domain: DomainCommon, Public: FieldUpdatedAt, Internal: FieldUpdatedAt},
	{Domain: DomainCommon, Public: FieldDeletedCount, Internal: FieldDeletedCount},
	{Domain: DomainCommon, Public: FieldExport, Internal: FieldExport},
	{Domain: DomainCommon, Public: FieldNamespace, Internal: FieldNamespace},
	{Domain: DomainCommon, Public: FieldEntries, Internal: FieldEntries},
	{Domain: DomainCommon, Public: FieldMacMasked, Internal: FieldMacMasked},
	{Domain: DomainCommon, Public: FieldPhoneMasked, Internal: FieldPhoneMasked},
	{Domain: DomainCommon, Public: FieldEmailMasked, Internal: FieldEmailMasked},
	{Domain: DomainCommon, Public: FieldSupportedBridgeType, Internal: FieldSupportedBridgeType},
	{Domain: DomainCommon, Public: FieldFirmwareVersion, Internal: FieldFirmwareVersion},
	{Domain: DomainCommon, Public: FieldFirmwareType, Internal: FieldFirmwareType},
	{Domain: DomainCommon, Public: FieldCurrentVersion, Internal: FieldCurrentVersion},
	{Domain: DomainCommon, Public: FieldTypeName, Internal: FieldTypeName},
	{Domain: DomainCommon, Public: FieldRoomRank, Internal: FieldRoomRank},
	{Domain: DomainCommon, Public: FieldRuleID, Internal: FieldRuleID},
	{Domain: DomainCommon, Public: FieldTimeInterval, Internal: FieldTimeInterval},
	{Domain: DomainCommon, Public: FieldVirtualDeviceCount, Internal: FieldVirtualDeviceCount},
	{Domain: DomainCommon, Public: FieldUnboundDevices, Internal: FieldUnboundDevices},
	{Domain: DomainCommon, Public: FieldUnboundGateways, Internal: FieldUnboundGateways},
	{Domain: DomainCommon, Public: FieldFeatures, Internal: FieldFeatures},
	{Domain: DomainCommon, Public: FieldValueRange, Internal: FieldValueRange},
	{Domain: DomainCommon, Public: FieldSupportActions, Internal: FieldSupportActions},
	{Domain: DomainCommon, Public: FieldQueryList, Internal: FieldQueryList},
	{Domain: DomainCommon, Public: FieldFAQItems, Internal: FieldFAQItems},
	{Domain: DomainCommon, Public: FieldPosition, Internal: FieldPosition},
	{Domain: DomainCommon, Public: FieldOrder, Internal: FieldOrder},
	{Domain: DomainCommon, Public: FieldOrderBy, Internal: FieldOrderBy},
	{Domain: DomainCommon, Public: FieldTotal, Internal: FieldTotal},
	{Domain: DomainCommon, Public: FieldCounts, Internal: FieldCounts},
	{Domain: DomainCommon, Public: FieldExpectedCounts, Internal: FieldExpectedCounts},
	{Domain: DomainCommon, Public: FieldObservedCounts, Internal: FieldObservedCounts},
	{Domain: DomainCommon, Public: FieldScript, Internal: FieldScript},
	{Domain: DomainCommon, Public: FieldResourceStatus, Internal: FieldResourceStatus},
	{Domain: DomainCommon, Public: FieldResourceIndex, Internal: FieldResourceIndex},
	{Domain: DomainCommon, Public: FieldResourceTypeID, Internal: FieldResourceTypeID},
	{Domain: DomainCommon, Public: FieldAttachments, Internal: FieldAttachments},
	{Domain: DomainCommon, Public: FieldResources, Internal: FieldResources},
	{Domain: DomainCommon, Public: FieldManualCandidateURL, Internal: FieldManualCandidateURL},
	{Domain: DomainCommon, Public: FieldFAQCandidateURL, Internal: FieldFAQCandidateURL},
	{Domain: DomainCommon, Public: FieldCandidateStatus, Internal: FieldCandidateStatus},
	{Domain: DomainCommon, Public: FieldManualAttachments, Internal: FieldManualAttachments},
	{Domain: DomainCommon, Public: FieldEncryption, Internal: FieldEncryption},
	{Domain: DomainCommon, Public: FieldImportPolicy, Internal: FieldImportPolicy},
	{Domain: DomainCommon, Public: FieldRetentionPolicy, Internal: FieldRetentionPolicy},
	{Domain: DomainCommon, Public: FieldConsents, Internal: FieldConsents},
	{Domain: DomainCommon, Public: FieldSignals, Internal: FieldSignals},
	{Domain: DomainCommon, Public: FieldSignalType, Internal: FieldSignalType},
	{Domain: DomainCommon, Public: FieldSignalKey, Internal: FieldSignalKey},
	{Domain: DomainCommon, Public: FieldFirstSeenAt, Internal: FieldFirstSeenAt},
	{Domain: DomainCommon, Public: FieldLastSeenAt, Internal: FieldLastSeenAt},
	{Domain: DomainCommon, Public: FieldLastShownAt, Internal: FieldLastShownAt},
	{Domain: DomainCommon, Public: FieldAccountProfile, Internal: FieldAccountProfile},
	{Domain: DomainCommon, Public: FieldProfile, Internal: FieldProfile},
	{Domain: DomainCommon, Public: FieldDataType, Internal: FieldDataType},
	{Domain: DomainCommon, Public: FieldLearningEnabled, Internal: FieldLearningEnabled},
	{Domain: DomainCommon, Public: FieldPaused, Internal: FieldPaused},
	{Domain: DomainCommon, Public: FieldConsentVersion, Internal: FieldConsentVersion},
	{Domain: DomainCommon, Public: FieldScopeType, Internal: FieldScopeType},
	{Domain: DomainCommon, Public: FieldScopeRef, Internal: FieldScopeRef},
	{Domain: DomainCommon, Public: FieldPreferenceID, Internal: FieldPreferenceID},
	{Domain: DomainCommon, Public: FieldPreferenceIDs, Internal: FieldPreferenceIDs},
	{Domain: DomainCommon, Public: FieldPreferenceType, Internal: FieldPreferenceType},
	{Domain: DomainCommon, Public: FieldPreferenceValue, Internal: FieldPreferenceValue},
	{Domain: DomainCommon, Public: FieldPreferences, Internal: FieldPreferences},
	{Domain: DomainCommon, Public: FieldExplicitPreferences, Internal: FieldExplicitPreferences},
	{Domain: DomainCommon, Public: FieldMemories, Internal: FieldMemories},
	{Domain: DomainCommon, Public: FieldRecommendation, Internal: FieldRecommendation},
	{Domain: DomainCommon, Public: FieldRecommendations, Internal: FieldRecommendations},
	{Domain: DomainCommon, Public: FieldRecommendationID, Internal: FieldRecommendationID},
	{Domain: DomainCommon, Public: FieldRecommendationIDs, Internal: FieldRecommendationIDs},
	{Domain: DomainCommon, Public: FieldRecommendationType, Internal: FieldRecommendationType},
	{Domain: DomainCommon, Public: FieldRecommendationEvidenceDays, Internal: FieldRecommendationEvidenceDays},
	{Domain: DomainCommon, Public: FieldRecommendationCompactionScope, Internal: FieldRecommendationCompactionScope},
	{Domain: DomainCommon, Public: FieldPendingRecommendations, Internal: FieldPendingRecommendations},
	{Domain: DomainCommon, Public: FieldFeedback, Internal: FieldFeedback},
	{Domain: DomainCommon, Public: FieldFeedbackRecorded, Internal: FieldFeedbackRecorded},
	{Domain: DomainCommon, Public: FieldCooldownHours, Internal: FieldCooldownHours},
	{Domain: DomainCommon, Public: FieldCooldownUntil, Internal: FieldCooldownUntil},
	{Domain: DomainCommon, Public: FieldSessionLimit, Internal: FieldSessionLimit},
	{Domain: DomainCommon, Public: FieldInteractionEventsDays, Internal: FieldInteractionEventsDays},
	{Domain: DomainCommon, Public: FieldInteractionEvidence, Internal: FieldInteractionEvidence},
	{Domain: DomainCommon, Public: FieldOperationLessonsRetention, Internal: FieldOperationLessonsRetention},
	{Domain: DomainCommon, Public: FieldRuntimeSubjectiveInferencePolicy, Internal: FieldRuntimeSubjectiveInferencePolicy},
	{Domain: DomainCommon, Public: FieldItem, Internal: FieldItem},
	{Domain: DomainCommon, Public: FieldKind, Internal: FieldKind},
	{Domain: DomainCommon, Public: FieldEvidence, Internal: FieldEvidence},
	{Domain: DomainCommon, Public: FieldExplanation, Internal: FieldExplanation},
	{Domain: DomainCommon, Public: FieldPriority, Internal: FieldPriority},
	{Domain: DomainCommon, Public: FieldConfidence, Internal: FieldConfidence},
	{Domain: DomainCommon, Public: FieldActionHint, Internal: FieldActionHint},
	{Domain: DomainCommon, Public: FieldActionSource, Internal: FieldActionSource},
	{Domain: DomainCommon, Public: FieldParametersHint, Internal: FieldParametersHint},
	{Domain: DomainCommon, Public: FieldTargetIntent, Internal: FieldTargetIntent},
	{Domain: DomainCommon, Public: FieldOperationIntent, Internal: FieldOperationIntent},
	{Domain: DomainCommon, Public: FieldLesson, Internal: FieldLesson},
	{Domain: DomainCommon, Public: FieldOperationLesson, Internal: FieldOperationLesson},
	{Domain: DomainCommon, Public: FieldOperationLessons, Internal: FieldOperationLessons},
	{Domain: DomainCommon, Public: FieldLessons, Internal: FieldLessons},
	{Domain: DomainCommon, Public: FieldLessonType, Internal: FieldLessonType},
	{Domain: DomainCommon, Public: FieldSymptom, Internal: FieldSymptom},
	{Domain: DomainCommon, Public: FieldCause, Internal: FieldCause},
	{Domain: DomainCommon, Public: FieldRecommendedPath, Internal: FieldRecommendedPath},
	{Domain: DomainCommon, Public: FieldAvoid, Internal: FieldAvoid},
	{Domain: DomainCommon, Public: FieldFallbackIntent, Internal: FieldFallbackIntent},
	{Domain: DomainCommon, Public: FieldStale, Internal: FieldStale},
	{Domain: DomainCommon, Public: FieldHitCount, Internal: FieldHitCount},
	{Domain: DomainCommon, Public: FieldLastValidatedAt, Internal: FieldLastValidatedAt},
	{Domain: DomainCommon, Public: FieldMinConfidence, Internal: FieldMinConfidence},
	{Domain: DomainCommon, Public: FieldConfidenceAtLeast, Internal: FieldConfidenceAtLeast},
	{Domain: DomainCommon, Public: FieldIncludeStale, Internal: FieldIncludeStale},
	{Domain: DomainCommon, Public: FieldIncludeRejected, Internal: FieldIncludeRejected},
	{Domain: DomainCommon, Public: FieldQuery, Internal: FieldQuery},
	{Domain: DomainCommon, Public: FieldQueryType, Internal: FieldQueryType},
	{Domain: DomainCommon, Public: FieldFuzzyName, Internal: FieldFuzzyName},
	{Domain: DomainCommon, Public: FieldExperience, Internal: FieldExperience},
	{Domain: DomainCommon, Public: FieldDelegatedIntent, Internal: FieldDelegatedIntent},
	{Domain: DomainCommon, Public: FieldTemporaryControl, Internal: FieldTemporaryControl},
	{Domain: DomainCommon, Public: FieldTestOnly, Internal: FieldTestOnly},
	{Domain: DomainAction, Public: FieldTargetType, Internal: internalTypeID},
	{Domain: DomainAction, Public: FieldTargetID, Internal: internalResourceID},
	{Domain: DomainAction, Public: FieldTargetKey, Internal: internalTempID},
	{Domain: DomainAction, Public: FieldTargetName, Internal: internalResourceName},
	{Domain: DomainAction, Public: FieldSubIndex, Internal: internalSubIndex},
	{Domain: DomainAction, Public: FieldActions, Internal: internalParams},
	{Domain: DomainAction, Public: FieldAction, Internal: FieldAction},
	{Domain: DomainAction, Public: FieldRank, Internal: FieldRank},
	{Domain: DomainAction, Public: FieldRoomID, Internal: FieldRoomID},
	{Domain: DomainAction, Public: FieldStartTime, Internal: FieldStartTime},
	{Domain: DomainAction, Public: FieldEndTime, Internal: FieldEndTime},
	{Domain: DomainAction, Public: FieldSet, Internal: FieldSet},
	{Domain: DomainAction, Public: FieldDelay, Internal: FieldDelay},
	{Domain: DomainAction, Public: FieldDuration, Internal: FieldDuration},
	{Domain: DomainAction, Public: FieldDelayOff, Internal: FieldDelayOff},
	{Domain: DomainAction, Public: FieldToggle, Internal: FieldToggle},
	{Domain: DomainAction, Public: FieldAdjust, Internal: FieldAdjust},
	{Domain: DomainAction, Public: FieldFlow, Internal: FieldFlow},
	{Domain: DomainAction, Public: FieldCustom, Internal: FieldCustom},
	{Domain: DomainAction, Public: FieldPower, Internal: internalPower},
	{Domain: DomainAction, Public: FieldBrightness, Internal: internalBrightness},
	{Domain: DomainAction, Public: FieldColorTemperature, Internal: internalColorTemperature},
	{Domain: DomainAction, Public: FieldColor, Internal: internalColor},
	{Domain: DomainAction, Public: FieldTargetPercent, Internal: internalTargetPercent},
	{Domain: DomainAction, Public: FieldSwitchPower, Internal: internalSwitchPower},
	{Domain: DomainAutomation, Public: FieldConditionType, Internal: internalConditionType},
	{Domain: DomainAutomation, Public: FieldConditionKind, Internal: internalConditionKind},
	{Domain: DomainAutomation, Public: FieldTime, Internal: internalClock},
	{Domain: DomainAutomation, Public: FieldTargetID, Internal: internalResourceID},
	{Domain: DomainAutomation, Public: FieldTargetKey, Internal: internalTempID},
	{Domain: DomainAutomation, Public: FieldTargetType, Internal: internalTypeID},
	{Domain: DomainAutomation, Public: FieldConditions, Internal: FieldConditions},
	{Domain: DomainAutomation, Public: FieldProperty, Internal: internalProperty},
	{Domain: DomainAutomation, Public: FieldOperation, Internal: FieldOperation},
	{Domain: DomainAutomation, Public: FieldOperators, Internal: FieldOperators},
	{Domain: DomainAutomation, Public: FieldValue, Internal: FieldValue},
	{Domain: DomainAutomation, Public: FieldCapabilityProductID, Internal: internalProductID},
	{Domain: DomainAutomation, Public: FieldEventID, Internal: FieldID},
	{Domain: DomainAutomation, Public: FieldEventArgs, Internal: internalEventArgs},
	{Domain: DomainProduct, Public: FieldSKUCode, Internal: internalProductCode},
	{Domain: DomainProduct, Public: FieldProductCode, Internal: internalProductCode},
	{Domain: DomainProduct, Public: FieldCapabilityProductID, Internal: internalProductID},
	{Domain: DomainProduct, Public: FieldCapabilityProductIDs, Internal: FieldCapabilityProductIDs},
	{Domain: DomainProduct, Public: FieldProductComponentID, Internal: internalProductCategoryID},
	{Domain: DomainProduct, Public: FieldProductCategoryID, Internal: internalProductCategoryID},
	{Domain: DomainProduct, Public: FieldProductName, Internal: FieldProductName},
	{Domain: DomainProduct, Public: FieldProductSKU, Internal: FieldProductSKU},
	{Domain: DomainProduct, Public: FieldProductSPU, Internal: FieldProductSPU},
	{Domain: DomainProduct, Public: FieldProductBrand, Internal: FieldProductBrand},
	{Domain: DomainProduct, Public: FieldProductModel, Internal: FieldProductModel},
	{Domain: DomainProduct, Public: FieldProductLine, Internal: FieldProductLine},
	{Domain: DomainProduct, Public: FieldProductCategory, Internal: FieldProductCategory},
	{Domain: DomainProduct, Public: FieldProductLargeClass, Internal: FieldProductLargeClass},
	{Domain: DomainProduct, Public: FieldProductSmallClass, Internal: FieldProductSmallClass},
	{Domain: DomainProduct, Public: FieldProductShortName, Internal: FieldProductShortName},
	{Domain: DomainProduct, Public: FieldProductSeries, Internal: FieldProductSeries},
	{Domain: DomainProduct, Public: FieldBarcode, Internal: FieldBarcode},
	{Domain: DomainProduct, Public: FieldBaseUnit, Internal: FieldBaseUnit},
	{Domain: DomainProduct, Public: FieldProductDeclareNo, Internal: FieldProductDeclareNo},
	{Domain: DomainProduct, Public: FieldProductDeclareName, Internal: FieldProductDeclareName},
	{Domain: DomainProduct, Public: FieldProductDeclareUnit, Internal: FieldProductDeclareUnit},
	{Domain: DomainProduct, Public: FieldSupportYeelightPro, Internal: FieldSupportYeelightPro},
	{Domain: DomainProduct, Public: FieldSupportHomeKit, Internal: FieldSupportHomeKit},
	{Domain: DomainProduct, Public: FieldProductStatusName, Internal: FieldProductStatusName},
	{Domain: DomainProduct, Public: FieldExtraMeta, Internal: FieldExtraMeta},
	{Domain: DomainProduct, Public: FieldModelNo, Internal: FieldModelNo},
	{Domain: DomainProduct, Public: FieldPediaDisplay, Internal: FieldPediaDisplay},
	{Domain: DomainProduct, Public: FieldProductSaleType, Internal: FieldProductSaleType},
	{Domain: DomainProduct, Public: FieldQuotationType, Internal: FieldQuotationType},
	{Domain: DomainProduct, Public: FieldProductTypeName, Internal: FieldProductTypeName},
	{Domain: DomainProduct, Public: FieldCategory, Internal: FieldCategory},
	{Domain: DomainProduct, Public: FieldSeries, Internal: FieldSeries},
	{Domain: DomainProduct, Public: FieldNotes, Internal: FieldNotes},
	{Domain: DomainProduct, Public: FieldConnectType, Internal: FieldConnectType},
	{Domain: DomainImport, Public: FieldKey, Internal: internalTempID},
	{Domain: DomainImport, Public: FieldRooms, Internal: internalRoomList},
	{Domain: DomainImport, Public: FieldDeviceSlots, Internal: internalDeviceList},
	{Domain: DomainImport, Public: FieldGroups, Internal: internalGroupList},
	{Domain: DomainImport, Public: FieldAreas, Internal: internalAreaList},
	{Domain: DomainImport, Public: FieldScenes, Internal: internalSceneList},
	{Domain: DomainImport, Public: FieldAutomations, Internal: internalAutomationList},
	{Domain: DomainImport, Public: FieldRoomKeys, Internal: internalRoomTempIDList},
	{Domain: DomainImport, Public: FieldSlotKeys, Internal: internalDeviceTempIDList},
	{Domain: DomainImport, Public: FieldComponentName, Internal: FieldComponentName},
	{Domain: DomainImport, Public: FieldProductEvidence, Internal: FieldProductEvidence},
	{Domain: DomainImport, Public: FieldGroupCategory, Internal: FieldGroupCategory},
	{Domain: DomainImport, Public: FieldGroupCapability, Internal: FieldGroupCapability},
	{Domain: DomainImport, Public: FieldCompatibleSlotKeys, Internal: FieldSlotKeys},
	{Domain: DomainImport, Public: FieldName, Internal: FieldName},
	{Domain: DomainImport, Public: FieldGateway, Internal: FieldGateway},
	{Domain: DomainImport, Public: FieldGatewayName, Internal: FieldName},
	{Domain: DomainImport, Public: FieldGatewayDeviceID, Internal: FieldGatewayDeviceID},
	{Domain: DomainImport, Public: FieldProduct, Internal: FieldProduct},
	{Domain: DomainImport, Public: FieldActions, Internal: FieldActions},
	{Domain: DomainImport, Public: FieldStartTime, Internal: FieldStartTime},
	{Domain: DomainImport, Public: FieldEndTime, Internal: FieldEndTime},
	{Domain: DomainPanel, Public: FieldID, Internal: FieldID},
	{Domain: DomainPanel, Public: FieldDeviceID, Internal: FieldDeviceID},
	{Domain: DomainPanel, Public: FieldButtons, Internal: FieldButtons},
	{Domain: DomainPanel, Public: FieldButtonEvent, Internal: FieldButtonEvent},
	{Domain: DomainPanel, Public: FieldButtonEventID, Internal: FieldButtonEventID},
	{Domain: DomainPanel, Public: FieldButtonEvents, Internal: FieldButtonEvents},
	{Domain: DomainPanel, Public: FieldButtonType, Internal: FieldButtonType},
	{Domain: DomainPanel, Public: FieldActions, Internal: FieldDetails},
	{Domain: DomainPanel, Public: FieldAlias, Internal: FieldAlias},
	{Domain: DomainPanel, Public: FieldKeyValue, Internal: FieldKeyValue},
	{Domain: DomainPanel, Public: FieldIndex, Internal: FieldIndex},
	{Domain: DomainPanel, Public: FieldRoomID, Internal: FieldRoomID},
	{Domain: DomainPanel, Public: FieldTargetID, Internal: internalResourceID},
	{Domain: DomainPanel, Public: FieldTargetType, Internal: internalResourceType},
	{Domain: DomainPanel, Public: FieldTargetName, Internal: internalResourceName},
	{Domain: DomainPanel, Public: FieldSubIndex, Internal: internalSubIndex},
	{Domain: DomainPanel, Public: FieldRank, Internal: FieldRank},
	{Domain: DomainPanel, Public: FieldVisible, Internal: FieldVisible},
	{Domain: DomainPanel, Public: FieldIcon, Internal: FieldIcon},
	{Domain: DomainPanel, Public: FieldSort, Internal: FieldSort},
	{Domain: DomainPanel, Public: FieldType, Internal: FieldType},
	{Domain: DomainPanel, Public: FieldExtend, Internal: FieldExtend},
	{Domain: DomainPanel, Public: FieldStartTime, Internal: FieldStartTime},
	{Domain: DomainPanel, Public: FieldEndTime, Internal: FieldEndTime},
	{Domain: DomainPanel, Public: FieldAction, Internal: FieldAction},
	{Domain: DomainPanel, Public: FieldProperty, Internal: FieldProperty},
	{Domain: DomainPanel, Public: FieldValue, Internal: FieldValue},
	{Domain: DomainPanel, Public: FieldDelay, Internal: FieldDelay},
	{Domain: DomainPanel, Public: FieldDuration, Internal: FieldDuration},
	{Domain: DomainKnob, Public: FieldID, Internal: FieldID},
	{Domain: DomainKnob, Public: FieldIndex, Internal: FieldIndex},
	{Domain: DomainKnob, Public: FieldActions, Internal: FieldDetails},
	{Domain: DomainKnob, Public: FieldConfigType, Internal: FieldConfigType},
	{Domain: DomainKnob, Public: FieldMode, Internal: FieldMode},
	{Domain: DomainKnob, Public: FieldModel, Internal: FieldModel},
	{Domain: DomainKnob, Public: FieldAction, Internal: FieldAction},
	{Domain: DomainKnob, Public: FieldProperty, Internal: FieldProperty},
	{Domain: DomainKnob, Public: FieldValue, Internal: FieldValue},
	{Domain: DomainKnob, Public: FieldTargetID, Internal: internalResourceID},
	{Domain: DomainKnob, Public: FieldTargetType, Internal: internalTypeID},
	{Domain: DomainKnob, Public: FieldTargetName, Internal: internalResourceName},
	{Domain: DomainKnob, Public: FieldSensitivity, Internal: internalSensitivity},
	{Domain: DomainFavorite, Public: FieldTargetID, Internal: internalResourceID},
	{Domain: DomainFavorite, Public: FieldTargetType, Internal: internalTypeID},
	{Domain: DomainSort, Public: FieldTargetID, Internal: internalResourceID},
	{Domain: DomainSort, Public: FieldTargetType, Internal: internalTypeID},
	{Domain: DomainHomeMember, Public: FieldExpiresAt, Internal: internalExpiresAt},
	{Domain: DomainHomeMember, Public: FieldCreateTime, Internal: FieldCreateTime},
	{Domain: DomainHomeMember, Public: FieldReuseBarcode, Internal: FieldReuseBarcode},
	{Domain: DomainHomeMember, Public: FieldShareID, Internal: FieldShareID},
	{Domain: DomainHomeMember, Public: FieldMemberID, Internal: FieldMemberID},
	{Domain: DomainHomeMember, Public: FieldMemberName, Internal: FieldMemberName},
	{Domain: DomainHomeMember, Public: FieldUserRole, Internal: FieldUserRole},
	{Domain: DomainHomeMember, Public: FieldRole, Internal: FieldRole},
	{Domain: DomainHomeMember, Public: FieldUID, Internal: FieldUID},
	{Domain: DomainHomeMember, Public: FieldUserID, Internal: FieldUserID},
	{Domain: DomainHomeMember, Public: FieldToUID, Internal: FieldToUID},
	{Domain: DomainPreview, Public: FieldScope, Internal: FieldScope},
	{Domain: DomainPreview, Public: FieldRisk, Internal: FieldRisk},
	{Domain: DomainPreview, Public: FieldIntent, Internal: FieldIntent},
	{Domain: DomainPreview, Public: FieldLocalOnly, Internal: FieldLocalOnly},
	{Domain: DomainPreview, Public: FieldHouseIndependent, Internal: FieldHouseIndependent},
	{Domain: DomainPreview, Public: FieldTargetEntityType, Internal: FieldTargetEntityType},
	{Domain: DomainPreview, Public: FieldTargetIDFlags, Internal: FieldTargetIDFlags},
	{Domain: DomainPreview, Public: FieldRequestSchema, Internal: FieldRequestSchema},
	{Domain: DomainPreview, Public: FieldPayloadGuide, Internal: FieldPayloadGuide},
	{Domain: DomainPreview, Public: FieldExampleCommand, Internal: FieldExampleCommand},
	{Domain: DomainPreview, Public: FieldIntentExplanation, Internal: FieldIntentExplanation},
	{Domain: DomainPreview, Public: FieldDirectFields, Internal: FieldDirectFields},
	{Domain: DomainPreview, Public: FieldAllowedIntents, Internal: FieldAllowedIntents},
	{Domain: DomainPreview, Public: FieldApplyIntent, Internal: FieldApplyIntent},
	{Domain: DomainPreview, Public: FieldApplyBehavior, Internal: FieldApplyBehavior},
	{Domain: DomainPreview, Public: FieldSummary, Internal: FieldSummary},
	{Domain: DomainPreview, Public: FieldExecutionModel, Internal: FieldExecutionModel},
	{Domain: DomainPreview, Public: FieldPreparedForDirectExecution, Internal: FieldPreparedForDirectExecution},
	{Domain: DomainPreview, Public: FieldPayloadPreview, Internal: FieldPayloadPreview},
	{Domain: DomainPreview, Public: FieldSemanticPreview, Internal: FieldSemanticPreview},
	{Domain: DomainPreview, Public: FieldPayload, Internal: FieldPayload},
	{Domain: DomainPreview, Public: FieldParameters, Internal: FieldParameters},
	{Domain: DomainPreview, Public: FieldTargets, Internal: FieldTargets},
	{Domain: DomainPreview, Public: FieldPreconditions, Internal: FieldPreconditions},
	{Domain: DomainPreview, Public: FieldDestructive, Internal: FieldDestructive},
	{Domain: DomainPreview, Public: FieldCurrent, Internal: FieldCurrent},
	{Domain: DomainPreview, Public: FieldCurrentItems, Internal: FieldCurrentItems},
	{Domain: DomainPreview, Public: FieldCurrentRoomID, Internal: FieldCurrentRoomID},
	{Domain: DomainPreview, Public: FieldPlanned, Internal: FieldPlanned},
	{Domain: DomainPreview, Public: FieldEntity, Internal: FieldEntity},
	{Domain: DomainPreview, Public: FieldDesign, Internal: FieldDesign},
	{Domain: DomainPreview, Public: FieldDeviceEvidence, Internal: FieldDeviceEvidence},
	{Domain: DomainPreview, Public: FieldCandidates, Internal: FieldCandidates},
	{Domain: DomainPreview, Public: FieldSupportedEntityTypes, Internal: FieldSupportedEntityTypes},
	{Domain: DomainPreview, Public: FieldMemberIDMasked, Internal: FieldMemberIDMasked},
	{Domain: DomainPreview, Public: FieldDisplayName, Internal: FieldDisplayName},
	{Domain: DomainPreview, Public: FieldImpact, Internal: FieldImpact},
	{Domain: DomainPreview, Public: FieldAffectedScope, Internal: FieldAffectedScope},
	{Domain: DomainPreview, Public: FieldCallerShouldConfirm, Internal: FieldCallerShouldConfirm},
	{Domain: DomainPreview, Public: FieldRuntimeApprovalStateStored, Internal: FieldRuntimeApprovalStateStored},
	{Domain: DomainPreview, Public: FieldItemCount, Internal: FieldItemCount},
	{Domain: DomainPreview, Public: FieldDeleteTargets, Internal: FieldDeleteTargets},
	{Domain: DomainPreview, Public: FieldDeleteTarget, Internal: FieldDeleteTarget},
	{Domain: DomainPreview, Public: FieldMatchedBy, Internal: FieldMatchedBy},
	{Domain: DomainPreview, Public: FieldFanOut, Internal: FieldFanOut},
	{Domain: DomainPreview, Public: FieldTargetMember, Internal: FieldTargetMember},
	{Domain: DomainPreview, Public: FieldBeforeValue, Internal: FieldBeforeValue},
	{Domain: DomainPreview, Public: FieldExpectedValue, Internal: FieldExpectedValue},
	{Domain: DomainPreview, Public: FieldVerified, Internal: FieldVerified},
	{Domain: DomainPreview, Public: FieldVerification, Internal: FieldVerification},
	{Domain: DomainPreview, Public: FieldVerifiedBy, Internal: FieldVerifiedBy},
	{Domain: DomainPreview, Public: FieldVerifiedValue, Internal: FieldVerifiedValue},
	{Domain: DomainPreview, Public: FieldVerifiedTopology, Internal: FieldVerifiedTopology},
	{Domain: DomainPreview, Public: FieldCommand, Internal: FieldCommand},
	{Domain: DomainPreview, Public: FieldAcceptedFields, Internal: FieldAcceptedFields},
	{Domain: DomainPreview, Public: FieldAcceptedValueFields, Internal: FieldAcceptedValueFields},
	{Domain: DomainPreview, Public: FieldTraceID, Internal: FieldTraceID},
	{Domain: DomainPreview, Public: FieldWarnings, Internal: FieldWarnings},
	{Domain: DomainPreview, Public: FieldUserMessage, Internal: FieldUserMessage},
	{Domain: DomainPreview, Public: FieldMessage, Internal: FieldMessage},
	{Domain: DomainPreview, Public: FieldError, Internal: FieldError},
	{Domain: DomainPreview, Public: FieldStepCount, Internal: FieldStepCount},
	{Domain: DomainPreview, Public: FieldSteps, Internal: FieldSteps},
	{Domain: DomainPreview, Public: FieldCompletedSteps, Internal: FieldCompletedSteps},
	{Domain: DomainPreview, Public: FieldFailedStep, Internal: FieldFailedStep},
	{Domain: DomainPreview, Public: FieldExclusions, Internal: FieldExclusions},
	{Domain: DomainPreview, Public: FieldWritePolicy, Internal: FieldWritePolicy},
	{Domain: DomainPreview, Public: FieldPreviewUnavailable, Internal: FieldPreviewUnavailable},
	{Domain: DomainPreview, Public: FieldWarning, Internal: FieldWarning},
	{Domain: DomainPreview, Public: FieldPlannedItems, Internal: FieldPlannedItems},
	{Domain: DomainPreview, Public: FieldPersistentWrites, Internal: FieldPersistentWrites},
	{Domain: DomainPreview, Public: FieldProductResolution, Internal: FieldProductResolution},
	{Domain: DomainPreview, Public: FieldCreatesDeviceSlots, Internal: FieldCreatesDeviceSlots},
	{Domain: DomainPreview, Public: FieldDeviceSlotsPhysical, Internal: FieldDeviceSlotsPhysical},
	{Domain: DomainPreview, Public: FieldTargetMode, Internal: FieldTargetMode},
	{Domain: DomainPreview, Public: FieldMatchedDeviceSlots, Internal: FieldMatchedDeviceSlots},
	{Domain: DomainPreview, Public: FieldUnresolvedDeviceSlots, Internal: FieldUnresolvedDeviceSlots},
	{Domain: DomainPreview, Public: FieldCatalog, Internal: FieldCatalog},
	{Domain: DomainPreview, Public: FieldSamples, Internal: FieldSamples},
	{Domain: DomainPreview, Public: FieldMappings, Internal: FieldMappings},
	{Domain: DomainPreview, Public: FieldRequestKey, Internal: FieldRequestKey},
	{Domain: DomainPreview, Public: FieldSelectedHouseID, Internal: FieldSelectedHouseID},
	{Domain: DomainPreview, Public: FieldResultData, Internal: FieldResultData},
	{Domain: DomainPreview, Public: FieldDeviceCount, Internal: FieldDeviceCount},
	{Domain: DomainPreview, Public: FieldChildDeviceCount, Internal: FieldChildDeviceCount},
	{Domain: DomainPreview, Public: FieldConfigs, Internal: FieldConfigs},
	{Domain: DomainPreview, Public: FieldConfigCount, Internal: FieldConfigCount},
	{Domain: DomainPreview, Public: FieldDeviceCountInRoom, Internal: FieldDeviceCountInRoom},
	{Domain: DomainPreview, Public: FieldRoomCount, Internal: FieldRoomCount},
	{Domain: DomainPreview, Public: FieldGroupCountInRoom, Internal: FieldGroupCountInRoom},
	{Domain: DomainPreview, Public: FieldRoomCountTotal, Internal: FieldRoomCountTotal},
	{Domain: DomainPreview, Public: FieldCreatedArtifacts, Internal: FieldCreatedArtifacts},
	{Domain: DomainPreview, Public: FieldActionCount, Internal: FieldActionCount},
	{Domain: DomainPreview, Public: FieldResults, Internal: FieldResults},
	{Domain: DomainPreview, Public: FieldStatusLabel, Internal: FieldStatusLabel},
	{Domain: DomainPreview, Public: FieldClearMAC, Internal: FieldClearMAC},
	{Domain: DomainPreview, Public: FieldUnbindRelatedDevices, Internal: FieldUnbindRelatedDevices},
	{Domain: DomainPreview, Public: FieldRecovery, Internal: FieldRecovery},
	{Domain: DomainPreview, Public: FieldSuggestedIntent, Internal: FieldSuggestedIntent},
	{Domain: DomainPreview, Public: FieldSafeNextStep, Internal: FieldSafeNextStep},
	{Domain: DomainPreview, Public: FieldCanRegenerate, Internal: FieldCanRegenerate},
	{Domain: DomainPreview, Public: FieldSafeToRetry, Internal: FieldSafeToRetry},
	{Domain: DomainPreview, Public: FieldTargetDeviceCount, Internal: FieldTargetDeviceCount},
	{Domain: DomainPreview, Public: FieldSupportedProperties, Internal: FieldSupportedProperties},
	{Domain: DomainPreview, Public: FieldSkipped, Internal: FieldSkipped},
	{Domain: DomainPreview, Public: FieldPolicyStatus, Internal: FieldPolicyStatus},
	{Domain: DomainPreview, Public: FieldHTTPStatus, Internal: FieldHTTPStatus},
	{Domain: DomainPreview, Public: FieldNextAction, Internal: FieldNextAction},
	{Domain: DomainMetrics, Public: FieldAPICalls, Internal: FieldAPICalls},
	{Domain: DomainMetrics, Public: FieldCacheHits, Internal: FieldCacheHits},
	{Domain: DomainMetrics, Public: FieldRuntimeMs, Internal: FieldRuntimeMs},
	{Domain: DomainMetrics, Public: FieldTopologyCacheRefreshCalls, Internal: FieldTopologyCacheRefreshCalls},
	{Domain: DomainMetrics, Public: FieldTopologyCacheWriteSource, Internal: FieldTopologyCacheWriteSource},
	{Domain: DomainState, Public: FieldRegion, Internal: FieldRegion},
	{Domain: DomainState, Public: FieldSource, Internal: FieldSource},
	{Domain: DomainState, Public: FieldQueryScope, Internal: FieldQueryScope},
	{Domain: DomainState, Public: FieldRawShape, Internal: FieldRawShape},
	{Domain: DomainState, Public: FieldPropertyName, Internal: FieldPropertyName},
	{Domain: DomainState, Public: FieldProperties, Internal: FieldProperties},
	{Domain: DomainState, Public: FieldSkippedProperties, Internal: FieldSkippedProperties},
	{Domain: DomainState, Public: FieldDiagnosticType, Internal: FieldDiagnosticType},
	{Domain: DomainState, Public: FieldStateSource, Internal: FieldStateSource},
	{Domain: DomainState, Public: FieldStateShape, Internal: FieldStateShape},
	{Domain: DomainState, Public: FieldExecutionIntent, Internal: FieldExecutionIntent},
	{Domain: DomainState, Public: FieldExecutionReadiness, Internal: FieldExecutionReadiness},
	{Domain: DomainState, Public: FieldExplanationScope, Internal: FieldExplanationScope},
	{Domain: DomainState, Public: FieldDetail, Internal: FieldDetail},
	{Domain: DomainState, Public: FieldCapabilitySource, Internal: FieldCapabilitySource},
	{Domain: DomainState, Public: FieldSchemaStatus, Internal: FieldSchemaStatus},
	{Domain: DomainState, Public: FieldDeviceSchema, Internal: FieldDeviceSchema},
	{Domain: DomainState, Public: FieldOperations, Internal: FieldOperations},
	{Domain: DomainState, Public: FieldLimitations, Internal: FieldLimitations},
	{Domain: DomainState, Public: FieldRead, Internal: FieldRead},
	{Domain: DomainState, Public: FieldWrite, Internal: FieldWrite},
	{Domain: DomainState, Public: FieldOnline, Internal: FieldOnline},
	{Domain: DomainState, Public: FieldBind, Internal: FieldBind},
	{Domain: DomainState, Public: FieldVirtual, Internal: FieldVirtual},
	{Domain: DomainState, Public: FieldPartial, Internal: FieldPartial},
	{Domain: DomainState, Public: FieldPartialState, Internal: FieldPartialState},
	{Domain: DomainState, Public: FieldTotal, Internal: FieldTotal},
	{Domain: DomainState, Public: FieldCounts, Internal: FieldCounts},
	{Domain: DomainState, Public: FieldExpectedCounts, Internal: FieldExpectedCounts},
	{Domain: DomainState, Public: FieldObservedCounts, Internal: FieldObservedCounts},
	{Domain: DomainState, Public: FieldEntities, Internal: FieldEntities},
	{Domain: DomainState, Public: FieldComponents, Internal: FieldComponents},
	{Domain: DomainState, Public: FieldEvents, Internal: FieldEvents},
	{Domain: DomainState, Public: FieldInputs, Internal: FieldInputs},
	{Domain: DomainState, Public: FieldAccess, Internal: FieldAccess},
	{Domain: DomainState, Public: FieldFormat, Internal: FieldFormat},
	{Domain: DomainState, Public: FieldUnit, Internal: FieldUnit},
	{Domain: DomainState, Public: FieldRange, Internal: FieldRange},
	{Domain: DomainState, Public: FieldMin, Internal: FieldMin},
	{Domain: DomainState, Public: FieldMax, Internal: FieldMax},
	{Domain: DomainState, Public: FieldValueList, Internal: FieldValueList},
	{Domain: DomainState, Public: FieldCode, Internal: FieldCode},
	{Domain: DomainState, Public: FieldEventID, Internal: FieldEventID},
	{Domain: DomainState, Public: FieldEventTypeID, Internal: FieldEventTypeID},
	{Domain: DomainState, Public: FieldEventNo, Internal: FieldEventNo},
	{Domain: DomainState, Public: FieldEventUnitNum, Internal: FieldEventUnitNum},
	{Domain: DomainState, Public: FieldNativeName, Internal: FieldNativeName},
	{Domain: DomainState, Public: FieldPropertyCount, Internal: FieldPropertyCount},
	{Domain: DomainState, Public: FieldComponentCount, Internal: FieldComponentCount},
	{Domain: DomainState, Public: FieldEventCount, Internal: FieldEventCount},
	{Domain: DomainState, Public: FieldReadable, Internal: FieldReadable},
	{Domain: DomainState, Public: FieldWritable, Internal: FieldWritable},
}

var publicToInternal = buildPublicToInternal(fieldMappings)

func buildPublicToInternal(mappings []FieldMapping) map[string]map[string]string {
	result := map[string]map[string]string{}
	for _, mapping := range mappings {
		if result[mapping.Domain] == nil {
			result[mapping.Domain] = map[string]string{}
		}
		result[mapping.Domain][mapping.Public] = mapping.Internal
	}
	return result
}

func InternalField(domain string, public string) string {
	if fields := publicToInternal[domain]; fields != nil {
		if value := fields[public]; value != "" {
			return value
		}
	}
	return public
}

func InternalActionParamsField() string {
	return internalParams
}

func InternalPanelActionParamsField() string {
	return internalParams
}

func InternalKnobActionParamsField() string {
	return internalParam
}

func InternalAutomationParamsField() string {
	return internalParams
}

func InternalProductIDsField() string {
	return internalProductIDs
}

func InternalCloudComponentIDField() string {
	return internalCloudComponentID
}

func InternalDeviceIdentifierField() string {
	return internalDeviceIdentifier
}

func InternalGroupCapabilityIDField() string {
	return internalComponentID
}

func InternalDeviceBindFlagField() string {
	return internalBindFlag
}

func InternalDeviceVirtualFlagField() string {
	return internalVirtualFlag
}

func InternalMetaImportStateField() string {
	return internalMetaImportState
}

func InternalUpperHouseIDField() string {
	return internalUpperHouseID
}

func InternalAccountIDCandidateFields() []string {
	return append([]string(nil), internalAccountIDCandidateFields...)
}

func InternalAccountDisplayNameCandidateFields() []string {
	return append([]string(nil), internalAccountDisplayNameCandidateFields...)
}

func InternalRepeatTypeField() string {
	return internalRepeatType
}

func InternalRepeatValueField() string {
	return internalRepeatValue
}

func ParameterPath(fields ...string) string {
	return FieldPath(append([]string{"parameters"}, fields...)...)
}

func FieldPath(fields ...string) string {
	return strings.Join(fields, ".")
}

func ArrayField(field string) string {
	return field + "[]"
}

func FieldRegistry() []PublicFieldMapping {
	result := make([]PublicFieldMapping, 0, len(fieldMappings))
	for _, mapping := range fieldMappings {
		result = append(result, PublicFieldMapping{
			Domain: mapping.Domain,
			Public: mapping.Public,
		})
	}
	return result
}
