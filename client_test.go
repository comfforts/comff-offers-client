package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	comffC "github.com/comfforts/comff-constants"
	offclient "github.com/comfforts/comff-offers-client"
	api "github.com/comfforts/comff-offers/api/v1"
	"github.com/comfforts/logger"
)

const TEST_DIR = "data"

func TestOffersClient(t *testing.T) {
	logger := logger.NewTestAppLogger(TEST_DIR)

	for scenario, fn := range map[string]func(
		t *testing.T,
		ofc offclient.Client,
	){
		"test database setup check, succeeds": testDatabaseSetup,
		"test offer CRUD, succeeds":           testOfferCRUD,
	} {
		t.Run(scenario, func(t *testing.T) {
			ofc, teardown := setup(t, logger)
			defer teardown()
			fn(t, ofc)
		})
	}

}

func setup(t *testing.T, logger logger.AppLogger) (
	dc offclient.Client,
	teardown func(),
) {
	t.Helper()

	clientOpts := offclient.NewDefaultClientOption()
	clientOpts.Caller = "offers-client-test"

	dc, err := offclient.NewClient(logger, clientOpts)
	require.NoError(t, err)

	return dc, func() {
		t.Logf(" %s ended, will clean up", t.Name())
		err := dc.Close()
		require.NoError(t, err)
	}
}

func testDatabaseSetup(t *testing.T, ofc offclient.Client) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	osResp, err := ofc.GetOfferStatuses(ctx, &api.OfferStatusesRequest{})
	require.NoError(t, err)
	require.Equal(t, len(osResp.Statuses), 10)

	dsResp, err := ofc.GetOfferTypes(ctx, &api.OfferTypesRequest{})
	require.NoError(t, err)
	require.Equal(t, len(dsResp.Types), 1)
}

func testOfferCRUD(t *testing.T, ofc offclient.Client) {
	t.Helper()

	reqtr, shopId, courierId, deliveryId := "test-client-offer-crud@gmail.com", "test-client-offer-crud-shop", "test-client-offer-crud-courier", "CL1eCr341e0r620ff3r"
	or := createOfferTester(t, ofc, &api.CreateOfferRequest{
		ActorId:       shopId,
		ParticipantId: courierId,
		TransactionId: deliveryId,
		RequestedBy:   reqtr,
		Min:           comffC.F12,
		Max:           comffC.F15,
	})
	or = getOfferTester(t, ofc, &api.GetOfferRequest{
		Id: or.Offer.Id,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := ofc.UpdateOffer(ctx, &api.UpdateOfferRequest{
		Id:          or.Offer.Id,
		Status:      api.OfferStatus_EXPIRED,
		RequestedBy: reqtr,
	})
	require.NoError(t, err)
	assert.Equal(t, resp.Offer.Status, api.OfferStatus_EXPIRED, "offer status should be EXPIRED")

	deleteOfferTester(t, ofc, &api.DeleteOfferRequest{
		Id: or.Offer.Id,
	})
}

func createOfferTester(t *testing.T, client offclient.Client, cor *api.CreateOfferRequest) *api.OfferResponse {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resp, err := client.CreateOffer(ctx, cor)
	require.NoError(t, err)
	assert.Equal(t, resp.Offer.ActorId, cor.ActorId, "offer actor id should match input actor id")
	assert.Equal(t, resp.Offer.ParticipantId, cor.ParticipantId, "offer participant id should match input participant id")
	assert.Equal(t, resp.Offer.TransactionId, cor.TransactionId, "offer transaction id should match input transaction id")
	assert.Equal(t, resp.Offer.Type, cor.Type, "offer type should match input type")
	assert.Equal(t, resp.Offer.Status, api.OfferStatus_OPEN, "offer status should be OPEN")

	assert.Equal(t, resp.Offer.Max, cor.Max, "offer max should match input max")
	assert.Equal(t, resp.Offer.Min, cor.Min, "offer min should match input min")

	return resp
}

func getOfferTester(t *testing.T, client offclient.Client, gor *api.GetOfferRequest) *api.OfferResponse {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := client.GetOffer(ctx, gor)
	require.NoError(t, err)
	require.Equal(t, resp.Offer.Id, gor.Id)
	return resp
}

func deleteOfferTester(t *testing.T, client offclient.Client, dor *api.DeleteOfferRequest) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := client.DeleteOffer(ctx, dor)
	require.NoError(t, err)
	require.Equal(t, true, resp.Ok)
}
