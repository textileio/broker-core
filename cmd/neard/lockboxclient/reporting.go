package lockboxclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
)

// DealInfo desscribes a single storage deal.
type DealInfo struct {
	DealID     uint64 `json:"dealId"`
	MinerID    string `json:"minerId"`
	Expiration uint64 `json:"expiration"`
}

// MarshalJSON implements MarshalJSON.
func (d DealInfo) MarshalJSON() ([]byte, error) {
	type Alias DealInfo
	return json.Marshal(&struct {
		DealID     string `json:"dealId"`
		Expiration string `json:"expiration"`
		Alias
	}{
		DealID:     strconv.FormatUint(d.DealID, 10),
		Expiration: strconv.FormatUint(d.Expiration, 10),
		Alias:      (Alias)(d),
	})
}

// UnmarshalJSON implements UnmarshalJSON.
func (d *DealInfo) UnmarshalJSON(data []byte) error {
	type Alias DealInfo
	aux := &struct {
		DealID     string `json:"dealId"`
		Expiration string `json:"expiration"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	dealID, err := strconv.ParseUint(aux.DealID, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing dealId: %v", err)
	}
	exp, err := strconv.ParseUint(aux.Expiration, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing expiration: %v", err)
	}
	d.DealID = dealID
	d.Expiration = exp
	return nil
}

// PayloadInfo describes something.
type PayloadInfo struct {
	PayloadCid string     `json:"payloadCid"`
	PieceCid   string     `json:"pieceCid"`
	Deals      []DealInfo `json:"deals"`
}

// PayloadOptions allow updating of the different parts of a storage payload.
type PayloadOptions struct {
	PieceCid string     `json:"pieceCid"`
	Deals    []DealInfo `json:"deals"`
	DataCids []string   `json:"dataCids"`
}

// ListPayloads lists payload records.
func (c *Client) ListPayloads(ctx context.Context, offset, maxLength int) ([]PayloadInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
		"listPayloads",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"offset": offset, "maxLength": maxLength}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	var infos []PayloadInfo
	if err := json.Unmarshal(res.Result, &infos); err != nil {
		return nil, fmt.Errorf("unmarshaling payload info: %v", err)
	}
	return infos, nil
}

// GetByPayload gets a payload record by payload cid.
func (c *Client) GetByPayload(ctx context.Context, payloadCid string) (*PayloadInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
		"getByPayload",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"payloadCid": payloadCid}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	if string(res.Result) == nullStringValue {
		return nil, ErrorNotFound
	}
	var payloadInfo PayloadInfo
	if err := json.Unmarshal(res.Result, &payloadInfo); err != nil {
		return nil, fmt.Errorf("unmarshaling payload info: %v", err)
	}
	return &payloadInfo, nil
}

// GetByCid gets payload records by data cid.
func (c *Client) GetByCid(ctx context.Context, dataCid string) ([]PayloadInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
		"getByCid",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"dataCid": dataCid}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	var payloadInfos []PayloadInfo
	if err := json.Unmarshal(res.Result, &payloadInfos); err != nil {
		return nil, fmt.Errorf("unmarshaling payload info: %v", err)
	}
	return payloadInfos, nil
}

// UpdatePayload creates or updates payload record.
func (c *Client) UpdatePayload(ctx context.Context, payloadCid string, options PayloadOptions) error {
	_, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.lockboxAccountID,
		"updatePayload",
		transaction.FunctionCallWithArgs(map[string]interface{}{
			"payloadCid": payloadCid,
			"options":    options,
		}),
	)
	if err != nil {
		return fmt.Errorf("calling rpc function: %v", err)
	}
	return nil
}
