package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	comffC "github.com/comfforts/comff-constants"
	offclient "github.com/comfforts/comff-offers-client"
	api "github.com/comfforts/comff-offers/api/v1"
	"github.com/comfforts/logger"
)

const TEST_DIR = "data"
const TEST_REQSTR = "test-offer-client@gmail.com"
const TEST_SHOP_ID = "test-offer-client-shop"
const TEST_COURIER_ID = "test-offer-client-courier"
const TEST_DELIVERY_ID = "CL1eCr341e0r620ff3r"
const TEST_WKFL_ID = "offer-client-test-wkflid"
const TEST_RUN_ID = "offer-client-test-wkflrunid"

func TestOffersClient(t *testing.T) {
	logger := logger.NewTestAppLogger(TEST_DIR)

	for scenario, fn := range map[string]func(
		t *testing.T,
		ofc offclient.Client,
	){
		"test database setup check, succeeds":    testDatabaseSetup,
		"test offer CRUD, succeeds":              testOfferCRUD,
		"duplicate offer test, succeeds":         testDuplicateOffer,
		"invalid offer creation check, succeeds": testInvalidOfferCreate,
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

	dur := (12 * time.Minute).Nanoseconds()
	or := createOfferTester(t, ofc, &api.CreateOfferRequest{
		ActorId:       TEST_SHOP_ID,
		ParticipantId: TEST_COURIER_ID,
		TransactionId: TEST_DELIVERY_ID,
		RequestedBy:   TEST_REQSTR,
		Min:           comffC.F12,
		Max:           comffC.F15,
		Duration:      dur,
		Distance:      comffC.F10,
		WorkflowId:    TEST_WKFL_ID,
		RunId:         TEST_RUN_ID,
	})
	require.Equal(t, float32(comffC.F10), or.Offer.Distance)
	require.Equal(t, dur, or.Offer.Duration)

	or = getOfferTester(t, ofc, &api.GetOfferRequest{
		Id: or.Offer.Id,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduleId := "t3s1Sch3d41e"
	resp, err := ofc.UpdateOffer(ctx, &api.UpdateOfferRequest{
		Id:          or.Offer.Id,
		Status:      api.OfferStatus_ACCEPT_PARTICIPANT,
		ScheduleId:  scheduleId,
		Value:       or.Offer.Max,
		Min:         or.Offer.Min,
		Max:         or.Offer.Max,
		RequestedBy: TEST_REQSTR,
	})
	require.NoError(t, err)
	assert.Equal(t, resp.Offer.Status, api.OfferStatus_ACCEPT_PARTICIPANT, "offer status should be ACCEPT_PARTICIPANT")
	assert.Equal(t, resp.Offer.ScheduleId, scheduleId, "offer schedule should match input schedule id")

	sResp, err := ofc.GetScheduleOffers(ctx, &api.GetOffersRequest{
		ScheduleId: scheduleId,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(sResp.Offers))
	require.Equal(t, scheduleId, sResp.Offers[0].ScheduleId)

	resp, err = ofc.UpdateOffer(ctx, &api.UpdateOfferRequest{
		Id:          or.Offer.Id,
		Status:      api.OfferStatus_EXPIRED,
		RequestedBy: TEST_REQSTR,
	})
	require.NoError(t, err)
	assert.Equal(t, resp.Offer.Status, api.OfferStatus_EXPIRED, "offer status should be EXPIRED")

	deleteOfferTester(t, ofc, &api.DeleteOfferRequest{
		Id: or.Offer.Id,
	})
}

func testInvalidOfferCreate(t *testing.T, ofc offclient.Client) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := ofc.CreateOffer(ctx, &api.CreateOfferRequest{
		ActorId:       TEST_SHOP_ID,
		ParticipantId: TEST_COURIER_ID,
		TransactionId: TEST_DELIVERY_ID,
		RequestedBy:   "",
		Min:           comffC.F12,
		Max:           comffC.F15,
		Duration:      (12 * time.Minute).Nanoseconds(),
		Distance:      comffC.F10,
		WorkflowId:    TEST_WKFL_ID,
		RunId:         TEST_RUN_ID,
	})
	require.Error(t, err)

	e, ok := status.FromError(err)
	require.Equal(t, ok, true)
	require.Equal(t, e.Code(), codes.InvalidArgument)
}

func testDuplicateOffer(t *testing.T, ofc offclient.Client) {
	t.Helper()

	or := createOfferTester(t, ofc, &api.CreateOfferRequest{
		ActorId:       TEST_SHOP_ID,
		ParticipantId: TEST_COURIER_ID,
		TransactionId: TEST_DELIVERY_ID,
		RequestedBy:   TEST_REQSTR,
		Min:           comffC.F12,
		Max:           comffC.F15,
		Duration:      (12 * time.Minute).Nanoseconds(),
		Distance:      comffC.F10,
		WorkflowId:    TEST_WKFL_ID,
		RunId:         TEST_RUN_ID,
	})
	or = getOfferTester(t, ofc, &api.GetOfferRequest{
		Id: or.Offer.Id,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := ofc.CreateOffer(ctx, &api.CreateOfferRequest{
		ActorId:       TEST_SHOP_ID,
		ParticipantId: TEST_COURIER_ID,
		TransactionId: TEST_DELIVERY_ID,
		RequestedBy:   TEST_REQSTR,
		Min:           comffC.F12,
		Max:           comffC.F15,
		Duration:      (12 * time.Minute).Nanoseconds(),
		Distance:      comffC.F10,
		WorkflowId:    TEST_WKFL_ID,
		RunId:         TEST_RUN_ID,
	})
	require.Error(t, err)

	e, ok := status.FromError(err)
	require.Equal(t, ok, true)
	require.Equal(t, e.Code(), codes.AlreadyExists)

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
	assert.Equal(t, resp.Offer.WorkflowId, cor.WorkflowId, "offer workflow id should match input workflow id")
	assert.Equal(t, resp.Offer.RunId, cor.RunId, "offer run id should match input run id")
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
