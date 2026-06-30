package storage

const (
	MemoryExportFormatVersion          = "yeelight-memory-export-v1"
	DefaultInteractionRetentionDays    = 90
	DefaultRecommendationRetentionDays = 90
	ExplicitPreferenceRetention        = "until_user_forgets"
	maxMergedEvidenceRunes             = 240
)

type PreferenceRecord struct {
	ID              string `json:"id"`
	Profile         string `json:"profile"`
	Region          string `json:"region,omitempty"`
	HouseID         string `json:"houseId"`
	ScopeType       string `json:"scopeType"`
	ScopeRef        string `json:"scopeRef"`
	PreferenceType  string `json:"preferenceType"`
	PreferenceValue string `json:"preferenceValue"`
	Kind            string `json:"kind,omitempty"`
	Status          string `json:"status,omitempty"`
	Evidence        string `json:"evidence,omitempty"`
	CreatedAt       int64  `json:"createdAt,omitempty"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type ConsentRecord struct {
	Profile         string `json:"profile"`
	Region          string `json:"region,omitempty"`
	HouseID         string `json:"houseId"`
	ConsentVersion  string `json:"consentVersion"`
	LearningEnabled bool   `json:"learningEnabled"`
	Paused          bool   `json:"paused"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type RecommendationRecord struct {
	ID             string         `json:"id"`
	Profile        string         `json:"profile"`
	Region         string         `json:"region,omitempty"`
	HouseID        string         `json:"houseId"`
	Type           string         `json:"type"`
	Source         string         `json:"source,omitempty"`
	TargetIntent   string         `json:"targetIntent,omitempty"`
	ScopeType      string         `json:"scopeType,omitempty"`
	ScopeRef       string         `json:"scopeRef,omitempty"`
	Priority       int            `json:"priority,omitempty"`
	Confidence     string         `json:"confidence,omitempty"`
	ActionHint     map[string]any `json:"actionHint,omitempty"`
	ParametersHint map[string]any `json:"parametersHint,omitempty"`
	Explanation    string         `json:"explanation"`
	Evidence       string         `json:"evidence"`
	Status         string         `json:"status"`
	CooldownUntil  int64          `json:"cooldownUntil,omitempty"`
	LastShownAt    int64          `json:"lastShownAt,omitempty"`
	CreatedAt      int64          `json:"createdAt"`
	UpdatedAt      int64          `json:"updatedAt"`
}

type RecommendationFeedback struct {
	Status        string
	CooldownUntil int64
	UpdatedAt     int64
}

type OperationLessonRecord struct {
	ID              string `json:"id"`
	Profile         string `json:"profile"`
	Region          string `json:"region,omitempty"`
	HouseID         string `json:"houseId,omitempty"`
	Intent          string `json:"intent"`
	LessonType      string `json:"lessonType"`
	Symptom         string `json:"symptom"`
	Cause           string `json:"cause,omitempty"`
	RecommendedPath string `json:"recommendedPath"`
	Avoid           string `json:"avoid,omitempty"`
	ParametersHint  string `json:"parametersHint,omitempty"`
	FallbackIntent  string `json:"fallbackIntent,omitempty"`
	Evidence        string `json:"evidence,omitempty"`
	Source          string `json:"source,omitempty"`
	Confidence      string `json:"confidence,omitempty"`
	Status          string `json:"status,omitempty"`
	Stale           bool   `json:"stale,omitempty"`
	HitCount        int    `json:"hitCount,omitempty"`
	LastValidatedAt int64  `json:"lastValidatedAt,omitempty"`
	CreatedAt       int64  `json:"createdAt,omitempty"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type JSONStore struct {
	path string
}

type PreferenceUpsertResult struct {
	Record  PreferenceRecord
	Created bool
	Merged  bool
}

type RecommendationUpsertResult struct {
	Record  RecommendationRecord
	Created bool
	Merged  bool
}

type OperationLessonUpsertResult struct {
	Record  OperationLessonRecord
	Created bool
	Merged  bool
}

type jsonDocument struct {
	Version         int                       `json:"version"`
	Namespace       StorageNamespace          `json:"namespace,omitempty"`
	Consents        []ConsentRecord           `json:"consents"`
	Preferences     []PreferenceRecord        `json:"preferences"`
	Recommendations []RecommendationRecord    `json:"recommendations"`
	Signals         []InteractionSignalRecord `json:"signals,omitempty"`
	Lessons         []OperationLessonRecord   `json:"lessons,omitempty"`
}

type StorageNamespace struct {
	AccountProfile string `json:"accountProfile,omitempty"`
	Profile        string `json:"profile"`
	Region         string `json:"region"`
	HouseID        string `json:"houseId"`
	DataType       string `json:"dataType"`
}

func NewJSONStore(path string) JSONStore {
	return JSONStore{path: path}
}
