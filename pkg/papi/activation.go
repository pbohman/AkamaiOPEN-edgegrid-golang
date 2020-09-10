package papi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/spf13/cast"
)

type (
	// ActivationFallbackInfo encapsulates information about fast fallback, which may allow you to fallback to a previous activation when
	// POSTing an activation with useFastFallback enabled.
	ActivationFallbackInfo struct {
		FastFallbackAttempted      bool   `json:"fastFallbackAttempted"`
		FallbackVersion            int    `json:"fallbackVersion"`
		CanFastFallback            bool   `json:"canFastFallback"`
		SteadyStateTime            int    `json:"steadyStateTime"`
		FastFallbackExpirationTime int    `json:"fastFallbackExpirationTime"`
		FastFallbackRecoveryState  string `json:"fastFallbackRecoveryState,omitempty"`
	}

	// Activation represents a property activation resource
	Activation struct {
		ActivationID           string                  `json:"activationId,omitempty"`
		ActivationType         ActivationType          `json:"activationType,omitempty"`
		FallbackInfo           *ActivationFallbackInfo `json:"fallbackInfo,omitempty"`
		AcknowledgeWarnings    []string                `json:"acknowledgeWarnings,omitempty"`
		AcknowledgeAllWarnings bool                    `json:"acknowledgeAllWarnings"`
		FastPush               bool                    `json:"fastPush,omitempty"`
		IgnoreHTTPErrors       bool                    `json:"ignoreHttpErrors,omitempty"`
		PropertyName           string                  `json:"propertyName,omitempty"`
		PropertyID             string                  `json:"propertyId,omitempty"`
		PropertyVersion        int                     `json:"propertyVersion"`
		Network                ActivationNetwork       `json:"network"`
		Status                 ActivationStatus        `json:"status,omitempty"`
		SubmitDate             string                  `json:"submitDate,omitempty"`
		UpdateDate             string                  `json:"updateDate,omitempty"`
		Note                   string                  `json:"note,omitempty"`
		NotifyEmails           []string                `json:"notifyEmails"`
	}

	// CreateActivationRequest is the request parameters for a new activation or deactivation request
	CreateActivationRequest struct {
		PropertyID string
		ContractID string
		GroupID    string
		Activation Activation
	}

	// CreateActivationResponse is the response for a new activation or deactivation
	CreateActivationResponse struct {
		ActivationID   string
		ActivationLink string `json:"activationLink"`
	}

	// GetActivationRequest is the get activation request
	GetActivationRequest struct {
		PropertyID   string
		ContractID   string
		GroupID      string
		ActivationID string
	}

	// GetActivationResponse is the get activation response
	GetActivationResponse struct {
		AccountID  string `json:"accountId"`
		ContractID string `json:"contractId"`
		GroupID    string `json:"groupId"`

		Activations struct {
			Items []*Activation `json:"items"`
		} `json:"contracts"`

		// RetryAfter is the value of the Retry-After header.
		//  For activations whose status is PENDING, a Retry-After header provides an estimate for when it’s likely to change.
		RetryAfter int `json:"-"`
	}

	// ActivationType is an activation type value
	ActivationType string

	// ActivationStatus is an activation status value
	ActivationStatus string

	// ActivationNetwork is the activation network value
	ActivationNetwork string
)

const (
	// ActivationTypeActivate is used for creating a new activation
	ActivationTypeActivate ActivationType = "ACTIVATE"

	// ActivationTypeDeactivate is used for creating a new de-activation
	ActivationTypeDeactivate ActivationType = "DEACTIVATE"

	// ActivationStatusActive is an activation that is currently serving traffic
	ActivationStatusActive ActivationStatus = "ACTIVE"

	// ActivationStatusInactive is an activation that has been superceded by another
	ActivationStatusInactive ActivationStatus = "INACTIVE"

	// ActivationStatusNew is a not yet active activation
	ActivationStatusNew ActivationStatus = "NEW"

	// ActivationStatusPending is the pending status
	ActivationStatusPending ActivationStatus = "PENDING"

	// ActivationStatusZone1 is not yet active
	ActivationStatusZone1 ActivationStatus = "ZONE_1"

	// ActivationStatusZone2 is not yet active
	ActivationStatusZone2 ActivationStatus = "ZONE_2"

	// ActivationStatusZone3 is not yet active
	ActivationStatusZone3 ActivationStatus = "ZONE_3"

	// ActivationStatusDeactivating is pending deactivation
	ActivationStatusDeactivating ActivationStatus = "PENDING_DEACTIVATION"

	// ActivationStatusDeactivated is deactivated
	ActivationStatusDeactivated ActivationStatus = "DEACTIVATED"

	// ActivationNetworkStaging is the staging network
	ActivationNetworkStaging ActivationNetwork = "STAGING"

	// ActivationNetworkProduction is the production network
	ActivationNetworkProduction ActivationNetwork = "PRODUCTION"
)

func (p *papi) CreateActivation(ctx context.Context, r CreateActivationRequest) (*CreateActivationResponse, error) {
	var rval CreateActivationResponse

	p.Log(ctx).Debug("CreateActivation")

	// explicitly set the activation type
	if r.Activation.ActivationType == "" {
		r.Activation.ActivationType = ActivationTypeActivate
	}

	uri := fmt.Sprintf("/papi/v1/properties/%s/activations?contractId=%s&groupId=%s", r.PropertyID, r.ContractID, r.GroupID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create createactivation request: %w", err)
	}

	req.Header.Set("PAPI-Use-Prefixes", cast.ToString(p.usePrefixes))

	resp, err := p.Exec(req, &rval, r.Activation)
	if err != nil {
		return nil, fmt.Errorf("createactivation request failed: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("createactivation request failed with status code: %d", resp.StatusCode)
	}

	return &rval, nil
}

func (p *papi) GetActivation(ctx context.Context, r GetActivationRequest) (*GetActivationResponse, error) {
	var rval GetActivationResponse

	p.Log(ctx).Debug("GetActivation")

	uri := fmt.Sprintf("/papi/v1/properties/%s/activations/%s?contractId=%s&groupId=%s", r.PropertyID, r.ActivationID, r.ContractID, r.GroupID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create getactivation request: %w", err)
	}

	req.Header.Set("PAPI-Use-Prefixes", cast.ToString(p.usePrefixes))

	resp, err := p.Exec(req, &rval)
	if err != nil {
		return nil, fmt.Errorf("getactivation request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getactivation request failed with status code: %d", resp.StatusCode)
	}

	// Get the Retry-After header to return the caller
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		rval.RetryAfter = cast.ToInt(retryAfter)
	}

	return &rval, nil
}
