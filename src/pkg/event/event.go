package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)


type SourceType string

const (
    SourceClinicalObservation SourceType = "clinical_observation"
    SourceSocialContact       SourceType = "social_contact"
    SourceFinancialEvent      SourceType = "financial_event"
    SourceRiskFlag            SourceType = "risk_flag"
    SourceCitizenStatement    SourceType = "citizen_statement"
    SourceExternalReference   SourceType = "external_reference"
    SourceCorrection          SourceType = "correction"
    SourceSystemEvent         SourceType = "system_event"
)

var validSources = map[SourceType]struct{}{
    SourceClinicalObservation: {},
    SourceSocialContact:       {},
    SourceFinancialEvent:      {},
    SourceRiskFlag:            {},
    SourceCitizenStatement:    {},
    SourceExternalReference:   {},
    SourceCorrection:          {},
    SourceSystemEvent:         {},
}

func (s SourceType) Valid() bool {
    _, ok := validSources[s]
    return ok
}



type EventType string

const (
    EventMedRoundCompleted EventType = "care.med_round_completed"
    EventSocialContactLog  EventType = "care.social_contact_log"
    EventRiskFlagRaised    EventType = "care.risk_flag_raised"
    EventCorrection        EventType = "care.correction"
    EventPensionPayout     EventType = "benefits.pension_payout"
    EventOverdueNotice     EventType = "benefits.overdue_notice"
    EventNoticeSuppressed  EventType = "benefits.notice_suppressed"
    EventSchemaMigration   EventType = "system.schema_migration"
)

func (e EventType) Subject() string {
    return "events." + string(e)
}

func (e EventType) Domain() string {
    for i, c := range e {
        if c == '.' {
            return string(e[:i])
        }
    }
    return string(e)
}

type ExternalReference struct {
    URL      string    `json:"url"`
    Platform string    `json:"platform"`
    Reason   string    `json:"reason"`
    LoggedAt time.Time `json:"logged_at"`
}

func (r *ExternalReference) Validate() error {
    var errs []error
    if r.URL == "" {
        errs = append(errs, errors.New("external_reference: URL is required"))
    }
    if r.Platform == "" {
        errs = append(errs, errors.New("external_reference: Platform is required"))
    }
    if len(r.Reason) < 20 {
        errs = append(errs, errors.New("external_reference: Reason must be at least 20 characters"))
    }
    if r.LoggedAt.IsZero() {
        errs = append(errs, errors.New("external_reference: LoggedAt is required"))
    }
    if len(errs) > 0 {
        return fmt.Errorf("external_reference validation: %v", errs)
    }
    return nil
}



type Correction struct {
    OriginalEventID uuid.UUID       `json:"original_event_id"`
    Reason          string          `json:"reason"`
    NewPayload      json.RawMessage `json:"new_payload"`
}

func (c *Correction) Validate() error {
    var errs []error
    if c.OriginalEventID == uuid.Nil {
        errs = append(errs, errors.New("correction: OriginalEventID is required"))
    }
    if len(c.Reason) < 10 {
        errs = append(errs, errors.New("correction: Reason must be at least 10 characters"))
    }
    if len(c.NewPayload) == 0 {
        errs = append(errs, errors.New("correction: NewPayload is required"))
    }
    if len(errs) > 0 {
        return fmt.Errorf("correction validation: %v", errs)
    }
    return nil
}



type Event struct {
    ID             uuid.UUID       `json:"id"`
    Type           EventType       `json:"type"`
    Source         SourceType      `json:"source"`
    Payload        json.RawMessage `json:"payload"`
    LoggedBy       string    `json:"logged_by"`
    LoggedFrom     string    `json:"logged_from"`
    EvidenceSource string    `json:"evidence_source"`
    ExternalRef    *ExternalReference `json:"external_ref,omitempty"`
    CorrectionRef  *uuid.UUID `json:"correction_ref,omitempty"`
    KommuneID      string    `json:"kommune_id"`
    IdempotencyKey string    `json:"idempotency_key"`
    Timestamp      time.Time `json:"timestamp"`
    SchemaVersion  int       `json:"schema_version"`
    TraceID        string    `json:"trace_id"`
}

func (e *Event) Validate() error {
    var errs []error

    if e.ID == uuid.Nil {
        errs = append(errs, errors.New("event: ID must not be nil"))
    }
    if e.Type == "" {
        errs = append(errs, errors.New("event: Type must not be empty"))
    }
    if !e.Source.Valid() {
        errs = append(errs, fmt.Errorf("event: invalid SourceType %q", e.Source))
    }
    if e.LoggedBy == "" {
        errs = append(errs, errors.New("event: LoggedBy must not be empty"))
    }
    if e.LoggedFrom == "" {
        errs = append(errs, errors.New("event: LoggedFrom must not be empty"))
    }
    if e.IdempotencyKey == "" {
        errs = append(errs, errors.New("event: IdempotencyKey must not be empty"))
    }
    if e.Timestamp.IsZero() {
        errs = append(errs, errors.New("event: Timestamp must not be zero"))
    }
    if e.KommuneID == "" {
        errs = append(errs, errors.New("event: KommuneID must not be empty (tenant isolation)"))
    }
    if e.SchemaVersion == 0 {
        errs = append(errs, errors.New("event: SchemaVersion must be > 0"))
    }

    if e.Source == SourceExternalReference {
        if e.ExternalRef == nil {
            errs = append(errs, errors.New("event: external_reference requires ExternalRef"))
        } else {
            if err := e.ExternalRef.Validate(); err != nil {
                errs = append(errs, err)
            }
        }
    }

    if e.Source == SourceCorrection {
        if e.CorrectionRef == nil {
            errs = append(errs, errors.New("event: correction requires CorrectionRef"))
        }
    }

  
    if e.Source == SourceClinicalObservation && e.ExternalRef != nil {
        errs = append(errs, errors.New(
            "event: clinical_observation cannot carry external_reference. "+
                "Use a separate event with SourceExternalReference.",
        ))
    }

    if e.Source == SourceFinancialEvent && e.ExternalRef != nil {
        errs = append(errs, errors.New(
            "event: financial_event cannot carry external_reference",
        ))
    }

    if len(errs) > 0 {
        return fmt.Errorf("event validation failed: %v", errs)
    }
    return nil
}


type MedicationRound struct {
    ResidentID      string    `json:"resident_id"`
    Medication      string    `json:"medication"`
    Deviation       string    `json:"deviation"`
    DeviationReason string    `json:"deviation_reason,omitempty"`
    CheckedBy       string    `json:"checked_by"`
    At              time.Time `json:"at"`
}

func (m *MedicationRound) Validate() error {
    var errs []error
    if m.ResidentID == "" {
        errs = append(errs, errors.New("medication_round: ResidentID required"))
    }
    if m.Medication == "" {
        errs = append(errs, errors.New("medication_round: Medication required"))
    }
    if m.CheckedBy == "" {
        errs = append(errs, errors.New("medication_round: CheckedBy required"))
    }
    if m.At.IsZero() {
        errs = append(errs, errors.New("medication_round: At required"))
    }
    validDeviations := map[string]struct{}{"none": {}, "skipped": {}, "extra_dose": {}, "delayed": {}}
    if _, ok := validDeviations[m.Deviation]; !ok {
        errs = append(errs, fmt.Errorf("medication_round: invalid deviation %q", m.Deviation))
    }
    if len(errs) > 0 {
        return fmt.Errorf("medication_round validation: %v", errs)
    }
    return nil
}

type SocialContact struct {
    CitizenID  string    `json:"citizen_id"`
    Caseworker string    `json:"caseworker"`
    Summary    string    `json:"summary"`
    At         time.Time `json:"at"`
}

func (s *SocialContact) Validate() error {
    var errs []error
    if s.CitizenID == "" {
        errs = append(errs, errors.New("social_contact: CitizenID required"))
    }
    if s.Caseworker == "" {
        errs = append(errs, errors.New("social_contact: Caseworker required"))
    }
    if len(s.Summary) < 10 {
        errs = append(errs, errors.New("social_contact: Summary must be at least 10 characters"))
    }
    if s.At.IsZero() {
        errs = append(errs, errors.New("social_contact: At required"))
    }
    if len(errs) > 0 {
        return fmt.Errorf("social_contact validation: %v", errs)
    }
    return nil
}

type PensionPayout struct {
    CitizenID string    `json:"citizen_id"`
    AmountDKK int64     `json:"amount_dkk"`
    Period    string    `json:"period"`
    Status    string    `json:"status"`
    At        time.Time `json:"at"`
}

func (p *PensionPayout) Validate() error {
    var errs []error
    if p.CitizenID == "" {
        errs = append(errs, errors.New("pension_payout: CitizenID required"))
    }
    if p.AmountDKK <= 0 {
        errs = append(errs, errors.New("pension_payout: AmountDKK must be positive"))
    }
    if p.Period == "" {
        errs = append(errs, errors.New("pension_payout: Period required"))
    }
    if p.At.IsZero() {
        errs = append(errs, errors.New("pension_payout: At required"))
    }
    validStatus := map[string]struct{}{"queued": {}, "paid": {}, "failed": {}, "reversed": {}}
    if _, ok := validStatus[p.Status]; !ok {
        errs = append(errs, fmt.Errorf("pension_payout: invalid status %q", p.Status))
    }
    if len(errs) > 0 {
        return fmt.Errorf("pension_payout validation: %v", errs)
    }
    return nil
}

type OverdueNotice struct {
    CitizenID     string    `json:"citizen_id"`
    CaseID        string    `json:"case_id"`
    AmountDKK     int64     `json:"amount_dkk"`
    IssuedAt      time.Time `json:"issued_at"`
    GraceDeadline time.Time `json:"grace_deadline"`
}

func (n *OverdueNotice) Validate() error {
    var errs []error
    if n.CitizenID == "" {
        errs = append(errs, errors.New("overdue_notice: CitizenID required"))
    }
    if n.CaseID == "" {
        errs = append(errs, errors.New("overdue_notice: CaseID required"))
    }
    if n.AmountDKK <= 0 {
        errs = append(errs, errors.New("overdue_notice: AmountDKK must be positive"))
    }
    if n.IssuedAt.IsZero() {
        errs = append(errs, errors.New("overdue_notice: IssuedAt required"))
    }
    expected := n.IssuedAt.Add(48 * time.Hour)
    if n.GraceDeadline != expected {
        errs = append(errs, fmt.Errorf(
            "overdue_notice: GraceDeadline must be IssuedAt + 48h, got %v", n.GraceDeadline,
        ))
    }
    if len(errs) > 0 {
        return fmt.Errorf("overdue_notice validation: %v", errs)
    }
    return nil
}

const CurrentSchemaVersion = 1

var SupportedSchemaVersions = map[int]struct{}{
    1: {},
}

func (e *Event) SchemaSupported() bool {
    _, ok := SupportedSchemaVersions[e.SchemaVersion]
    return ok
}