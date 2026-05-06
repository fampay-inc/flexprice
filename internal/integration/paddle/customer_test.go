package paddle

import (
	"context"
	"errors"
	"testing"

	paddlesdk "github.com/PaddleHQ/paddle-go-sdk/v4"
	"github.com/flexprice/flexprice/internal/domain/connection"
	"github.com/flexprice/flexprice/internal/domain/customer"
	"github.com/flexprice/flexprice/internal/domain/entityintegrationmapping"
	"github.com/flexprice/flexprice/internal/logger"
	"github.com/flexprice/flexprice/internal/types"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakePaddleClient struct {
	createAddressFn func(ctx context.Context, customerID string, req *paddlesdk.CreateAddressRequest) (*paddlesdk.Address, error)
	updateAddressFn func(ctx context.Context, customerID string, addressID string, req *paddlesdk.UpdateAddressRequest) (*paddlesdk.Address, error)
}

// Implement all PaddleClient interface methods with exact return types
func (f *fakePaddleClient) GetPaddleConfig(_ context.Context) (*PaddleConfig, error) {
	return nil, nil
}
func (f *fakePaddleClient) GetDecryptedPaddleConfig(_ *connection.Connection) (*PaddleConfig, error) {
	return nil, nil
}
func (f *fakePaddleClient) HasPaddleConnection(_ context.Context) bool {
	return true
}
func (f *fakePaddleClient) GetConnection(_ context.Context) (*connection.Connection, error) {
	return nil, nil
}
func (f *fakePaddleClient) GetSDKClient(_ context.Context) (*paddlesdk.SDK, *PaddleConfig, error) {
	return nil, nil, nil
}
func (f *fakePaddleClient) CreateCustomer(_ context.Context, _ *paddlesdk.CreateCustomerRequest) (*paddlesdk.Customer, error) {
	return nil, nil
}
func (f *fakePaddleClient) CreateAddress(ctx context.Context, customerID string, req *paddlesdk.CreateAddressRequest) (*paddlesdk.Address, error) {
	if f.createAddressFn != nil {
		return f.createAddressFn(ctx, customerID, req)
	}
	return &paddlesdk.Address{ID: "adr_new"}, nil
}
func (f *fakePaddleClient) UpdateAddress(ctx context.Context, customerID string, addressID string, req *paddlesdk.UpdateAddressRequest) (*paddlesdk.Address, error) {
	if f.updateAddressFn != nil {
		return f.updateAddressFn(ctx, customerID, addressID, req)
	}
	return &paddlesdk.Address{ID: addressID}, nil
}
func (f *fakePaddleClient) CreateTransaction(_ context.Context, _ *paddlesdk.CreateTransactionRequest) (*paddlesdk.Transaction, error) {
	return nil, nil
}
func (f *fakePaddleClient) PreviewTransaction(_ context.Context, _ *paddlesdk.PreviewTransactionCreateRequest) (*paddlesdk.TransactionPreview, error) {
	return nil, nil
}
func (f *fakePaddleClient) VerifyWebhookSignature(_ context.Context, _ []byte, _, _ string) error {
	return nil
}

type memMappingRepo struct {
	rows map[string]*entityintegrationmapping.EntityIntegrationMapping
}

func newMemMappingRepo() *memMappingRepo {
	return &memMappingRepo{rows: make(map[string]*entityintegrationmapping.EntityIntegrationMapping)}
}

func (m *memMappingRepo) Create(_ context.Context, mapping *entityintegrationmapping.EntityIntegrationMapping) error {
	m.rows[mapping.ID] = mapping
	return nil
}

func (m *memMappingRepo) Get(_ context.Context, id string) (*entityintegrationmapping.EntityIntegrationMapping, error) {
	if row, ok := m.rows[id]; ok {
		return row, nil
	}
	return nil, errors.New("not found")
}

func (m *memMappingRepo) List(_ context.Context, filter *types.EntityIntegrationMappingFilter) ([]*entityintegrationmapping.EntityIntegrationMapping, error) {
	var out []*entityintegrationmapping.EntityIntegrationMapping
	for _, row := range m.rows {
		if filter.EntityID != "" && row.EntityID != filter.EntityID {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (m *memMappingRepo) Count(_ context.Context, _ *types.EntityIntegrationMappingFilter) (int, error) {
	return len(m.rows), nil
}

func (m *memMappingRepo) Update(_ context.Context, mapping *entityintegrationmapping.EntityIntegrationMapping) error {
	m.rows[mapping.ID] = mapping
	return nil
}

func (m *memMappingRepo) Delete(_ context.Context, mapping *entityintegrationmapping.EntityIntegrationMapping) error {
	delete(m.rows, mapping.ID)
	return nil
}

func mustTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	return logger.NewNoopLogger()
}

// --- tests ---

func TestSyncPaddleAddress(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, types.CtxTenantID, "tenant_test")
	ctx = context.WithValue(ctx, types.CtxEnvironmentID, "env_test")
	ctx = context.WithValue(ctx, types.CtxUserID, "user_test")

	const paddleCustomerID = "ctm_test"
	const paddleAddressID = "adr_existing"

	tests := []struct {
		name             string
		customer         *customer.Customer
		existingMapping  *entityintegrationmapping.EntityIntegrationMapping
		createAddressFn  func(ctx context.Context, customerID string, req *paddlesdk.CreateAddressRequest) (*paddlesdk.Address, error)
		updateAddressFn  func(ctx context.Context, customerID string, addressID string, req *paddlesdk.UpdateAddressRequest) (*paddlesdk.Address, error)
		wantErr          bool
		wantCreateCalled bool
		wantUpdateCalled bool
		wantAddressIDIn  string // expected paddle_address_id stored in mapping after call
	}{
		{
			name: "no_country_is_noop",
			customer: &customer.Customer{
				ID:    "cust_1",
				Email: "a@b.com",
				// AddressCountry intentionally empty
			},
			wantErr:          false,
			wantCreateCalled: false,
			wantUpdateCalled: false,
		},
		{
			name: "no_existing_address_creates_and_stores",
			customer: &customer.Customer{
				ID:                "cust_2",
				Email:             "a@b.com",
				AddressCountry:    "US",
				AddressLine1:      "123 Main St",
				AddressCity:       "New York",
				AddressState:      "NY",
				AddressPostalCode: "10001",
			},
			existingMapping: &entityintegrationmapping.EntityIntegrationMapping{
				ID:               "map_1",
				EntityID:         "cust_2",
				EntityType:       types.IntegrationEntityTypeCustomer,
				ProviderType:     string(types.SecretProviderPaddle),
				ProviderEntityID: paddleCustomerID,
				Metadata:         map[string]interface{}{}, // no paddle_address_id
			},
			wantErr:          false,
			wantCreateCalled: true,
			wantUpdateCalled: false,
			wantAddressIDIn:  "adr_new",
		},
		{
			name: "existing_address_calls_update",
			customer: &customer.Customer{
				ID:                "cust_3",
				Email:             "a@b.com",
				AddressCountry:    "GB",
				AddressLine1:      "10 Downing St",
				AddressCity:       "London",
				AddressPostalCode: "SW1A 2AA",
			},
			existingMapping: &entityintegrationmapping.EntityIntegrationMapping{
				ID:               "map_2",
				EntityID:         "cust_3",
				EntityType:       types.IntegrationEntityTypeCustomer,
				ProviderType:     string(types.SecretProviderPaddle),
				ProviderEntityID: paddleCustomerID,
				Metadata:         map[string]interface{}{"paddle_address_id": paddleAddressID},
			},
			wantErr:          false,
			wantCreateCalled: false,
			wantUpdateCalled: true,
			wantAddressIDIn:  paddleAddressID, // ID unchanged after update
		},
		{
			name: "update_paddle_error_soft_fails",
			customer: &customer.Customer{
				ID:             "cust_4",
				Email:          "a@b.com",
				AddressCountry: "US",
			},
			existingMapping: &entityintegrationmapping.EntityIntegrationMapping{
				ID:               "map_3",
				EntityID:         "cust_4",
				EntityType:       types.IntegrationEntityTypeCustomer,
				ProviderType:     string(types.SecretProviderPaddle),
				ProviderEntityID: paddleCustomerID,
				Metadata:         map[string]interface{}{"paddle_address_id": paddleAddressID},
			},
			updateAddressFn: func(_ context.Context, _, _ string, _ *paddlesdk.UpdateAddressRequest) (*paddlesdk.Address, error) {
				return nil, errors.New("paddle API error")
			},
			wantErr:          false, // soft-fail: no error returned
			wantUpdateCalled: true,
		},
		{
			name: "create_paddle_error_hard_fails",
			customer: &customer.Customer{
				ID:             "cust_5",
				Email:          "a@b.com",
				AddressCountry: "US",
			},
			existingMapping: &entityintegrationmapping.EntityIntegrationMapping{
				ID:               "map_4",
				EntityID:         "cust_5",
				EntityType:       types.IntegrationEntityTypeCustomer,
				ProviderType:     string(types.SecretProviderPaddle),
				ProviderEntityID: paddleCustomerID,
				Metadata:         map[string]interface{}{}, // no paddle_address_id
			},
			createAddressFn: func(_ context.Context, _ string, _ *paddlesdk.CreateAddressRequest) (*paddlesdk.Address, error) {
				return nil, errors.New("paddle API error")
			},
			wantErr:          true,
			wantCreateCalled: true,
		},
		{
			name: "no_mapping_row_creates_address_and_mapping",
			customer: &customer.Customer{
				ID:             "cust_6",
				Email:          "a@b.com",
				AddressCountry: "DE",
				AddressCity:    "Berlin",
			},
			// no existingMapping — repo starts empty
			wantErr:          false,
			wantCreateCalled: true,
			wantAddressIDIn:  "adr_new",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			createCalled := false
			updateCalled := false

			fakeClient := &fakePaddleClient{
				createAddressFn: func(ctx context.Context, customerID string, req *paddlesdk.CreateAddressRequest) (*paddlesdk.Address, error) {
					createCalled = true
					if tc.createAddressFn != nil {
						return tc.createAddressFn(ctx, customerID, req)
					}
					return &paddlesdk.Address{ID: "adr_new"}, nil
				},
				updateAddressFn: func(ctx context.Context, customerID string, addressID string, req *paddlesdk.UpdateAddressRequest) (*paddlesdk.Address, error) {
					updateCalled = true
					if tc.updateAddressFn != nil {
						return tc.updateAddressFn(ctx, customerID, addressID, req)
					}
					return &paddlesdk.Address{ID: addressID}, nil
				},
			}

			repo := newMemMappingRepo()
			if tc.existingMapping != nil {
				repo.rows[tc.existingMapping.ID] = tc.existingMapping
			}

			svc := &CustomerService{
				client:                       fakeClient,
				entityIntegrationMappingRepo: repo,
				logger:                       mustTestLogger(t),
			}

			err := svc.syncPaddleAddress(ctx, tc.customer, paddleCustomerID)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.wantCreateCalled, createCalled, "CreateAddress call mismatch")
			require.Equal(t, tc.wantUpdateCalled, updateCalled, "UpdateAddress call mismatch")

			if tc.wantAddressIDIn != "" {
				mappings, listErr := repo.List(ctx, &types.EntityIntegrationMappingFilter{
					EntityID: tc.customer.ID,
				})
				require.NoError(t, listErr)
				require.NotEmpty(t, mappings, "expected a mapping row to exist")
				addrID, _ := mappings[0].Metadata["paddle_address_id"].(string)
				require.Equal(t, tc.wantAddressIDIn, addrID, "paddle_address_id in mapping mismatch")
			}
		})
	}
}
