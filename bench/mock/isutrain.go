package mock

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/chibiegg/isucon9-final/bench/internal/util"
	"github.com/chibiegg/isucon9-final/bench/isutrain"
	"github.com/gorilla/sessions"
	"github.com/jarcoal/httpmock"
	"net/http/httptest"
)

// Mock は `isutrain` のモック実装です
type Mock struct {
	LoginDelay             time.Duration
	ListStationsDelay      time.Duration
	SearchTrainsDelay      time.Duration
	ListTrainSeatsDelay    time.Duration
	ReserveDelay           time.Duration
	CommitReservationDelay time.Duration
	CancelReservationDelay time.Duration
	ListReservationDelay   time.Duration

	sessionName string
	session     sessions.Store

	injectFunc func(path string) error

	paymentMock *paymentMock
}

func NewMock(paymentMock *paymentMock) (*Mock, error) {
	randomStr, err := util.SecureRandomStr(20)
	if err != nil {
		return nil, err
	}
	return &Mock{
		injectFunc: func(path string) error {
			return nil
		},
		paymentMock: paymentMock,
		sessionName: "session_isutrain",
		session:     sessions.NewCookieStore([]byte(randomStr)),
	}, nil
}

func (m *Mock) getSession(req *http.Request) (*sessions.Session, error) {
	session, err := m.session.Get(req, m.sessionName)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (m *Mock) Inject(f func(path string) error) {
	m.injectFunc = f
}

func (m *Mock) Initialize(req *http.Request) ([]byte, int) {
	if err := m.injectFunc(req.URL.Path); err != nil {
		return []byte(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError
	}
	return []byte(http.StatusText(http.StatusAccepted)), http.StatusAccepted
}

// Register はユーザ登録を行います
func (m *Mock) Register(req *http.Request) ([]byte, int) {
	if err := req.ParseForm(); err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	var (
		username = req.Form.Get("email")
		password = req.Form.Get("password")
	)
	if len(username) == 0 || len(password) == 0 {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	return []byte(http.StatusText(http.StatusAccepted)), http.StatusAccepted
}

// Login はログイン処理結果を返します
func (m *Mock) Login(req *http.Request) ([]byte, int) {
	<-time.After(m.LoginDelay)
	if err := req.ParseForm(); err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	var (
		username = req.Form.Get("username")
		password = req.Form.Get("password")
	)
	if len(username) == 0 || len(password) == 0 {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	session, err := m.getSession(req)
	if err != nil {
		return []byte(http.StatusText(http.StatusNotFound)), http.StatusNotFound
	}
	session.Values["user_id"] = 1
	session.Values["csrf_token"], err = util.SecureRandomStr(20)

	if err != nil {
		return []byte(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError
	}

	wr := httptest.NewRecorder()


	return []byte(http.StatusText(http.StatusAccepted)), http.StatusAccepted
}

func (m *Mock) ListStations(req *http.Request) ([]byte, int) {
	<-time.After(m.ListStationsDelay)
	b, err := json.Marshal([]*isutrain.Station{
		&isutrain.Station{ID: 1, Name: "isutrain1", IsStopExpress: false, IsStopSemiExpress: false, IsStopLocal: false},
	})
	if err != nil {
		return []byte(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError
	}

	return b, http.StatusOK
}

// SearchTrains は新幹線検索結果を返します
func (m *Mock) SearchTrains(req *http.Request) ([]byte, int) {
	<-time.After(m.SearchTrainsDelay)
	query := req.URL.Query()
	if query == nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	// TODO: 検索クエリを受け取る
	// いつ(use_at)、どこからどこまで(from, to), 人数(number of people) で結果が帰って来れば良い
	// 日付を投げてきていて、DB称号可能などこからどこまでがあればいい
	// どこからどこまでは駅名を書く(IDはユーザから見たらまだわからない)
	useAt, err := util.ParseISO8601(query.Get("use_at"))
	if err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}
	if useAt.IsZero() {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	var (
		from = query.Get("from")
		to   = query.Get("to")
	)
	if len(from) == 0 || len(to) == 0 {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	var (
		seatAvailability = map[string]string{
			string(isutrain.SaPremium):       "○",
			string(isutrain.SaPremiumSmoke):  "×",
			string(isutrain.SaReserved):      "△",
			string(isutrain.SaReservedSmoke): "○",
			string(isutrain.SaNonReserved):   "○",
		}
		fareInformation = map[string]int{
			string(isutrain.SaPremium):       24000,
			string(isutrain.SaPremiumSmoke):  24500,
			string(isutrain.SaReserved):      19000,
			string(isutrain.SaReservedSmoke): 19500,
			string(isutrain.SaNonReserved):   15000,
		}
	)
	b, err := json.Marshal(&isutrain.Trains{
		&isutrain.Train{
			Class:            "のぞみ",
			Name:             "96号",
			Start:            1,
			Last:             2,
			Departure:        "東京",
			Destination:      "名古屋",
			DepartedAt:       time.Now(),
			ArrivedAt:        time.Now(),
			SeatAvailability: seatAvailability,
			FareInformation:  fareInformation,
		},
		&isutrain.Train{
			Class:            "こだま",
			Name:             "96号",
			Start:            3,
			Last:             4,
			Departure:        "名古屋",
			Destination:      "大阪",
			DepartedAt:       time.Now(),
			ArrivedAt:        time.Now(),
			SeatAvailability: seatAvailability,
			FareInformation:  fareInformation,
		},
	})
	if err != nil {
		return []byte(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError
	}

	return b, http.StatusOK
}

// ListTrainSeats は列車の席一覧を返します
func (m *Mock) ListTrainSeats(req *http.Request) ([]byte, int) {
	<-time.After(m.ListTrainSeatsDelay)
	q := req.URL.Query()
	if q == nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	// 列車特定情報を受け取る
	var (
		trainClass   = q.Get("train_class")
		trainName    = q.Get("train_name")
		carNumber, _ = strconv.Atoi(q.Get("car_number"))
		fromName     = q.Get("from")
		toName       = q.Get("to")
	)
	if len(trainClass) == 0 || len(trainName) == 0 {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}
	if len(fromName) == 0 || len(toName) == 0 {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}
	if carNumber == 0 {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	// 適当な席を返す
	b, err := json.Marshal(&isutrain.TrainSeatSearchResponse{
		UseAt:      time.Now(),
		TrainClass: "dummy",
		TrainName:  "dummy",
		CarNumber:  1,
		Seats: isutrain.TrainSeats{
			&isutrain.TrainSeat{},
		},
	})
	if err != nil {
		return []byte(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError
	}

	return b, http.StatusOK
}

// Reserve は座席予約を実施し、結果を返します
func (m *Mock) Reserve(req *http.Request) ([]byte, int) {
	<-time.After(m.ReserveDelay)
	// 予約情報を受け取って、予約できたかを返す
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	// 複数の座席指定で予約するかもしれない
	// なので、予約には複数の座席予約が紐づいている
	var reservationReq *isutrain.ReservationRequest
	if err := json.Unmarshal(b, &reservationReq); err != nil {
		log.Println("unmarshal fail")
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	if len(reservationReq.TrainClass) == 0 || len(reservationReq.TrainName) == 0 {
		log.Println("train info fail")
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	b, err = json.Marshal(&isutrain.ReservationResponse{
		ReservationID: "1111111111",
		IsOk:          true,
	})
	if err != nil {
		return []byte(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError
	}

	// NOTE: とりあえず、パラメータガン無視でPOSTできるところ先にやる
	return b, http.StatusAccepted
}

// CommitReservation は予約を確定します
func (m *Mock) CommitReservation(req *http.Request) ([]byte, int) {
	<-time.After(m.CommitReservationDelay)
	// 予約IDを受け取って、確定するだけ

	_, err := httpmock.GetSubmatchAsUint(req, 1)
	if err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	// FIXME: ちゃんとした決済情報を追加する
	m.paymentMock.addPaymentInformation()

	return []byte(http.StatusText(http.StatusAccepted)), http.StatusAccepted
}

// CancelReservation は予約をキャンセルします
func (m *Mock) CancelReservation(req *http.Request) ([]byte, int) {
	<-time.After(m.CancelReservationDelay)
	// 予約IDを受け取って

	_, err := httpmock.GetSubmatchAsUint(req, 1)
	if err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	return []byte(http.StatusText(http.StatusNoContent)), http.StatusNoContent
}

// ListReservations はアカウントにひもづく予約履歴を返します
func (m *Mock) ListReservations(req *http.Request) ([]byte, int) {
	<-time.After(m.ListReservationDelay)
	b, err := json.Marshal(isutrain.SeatReservations{
		&isutrain.SeatReservation{ID: 1111, PaymentMethod: string(isutrain.CreditCard), Status: string(isutrain.Pending), ReserveAt: time.Now()},
	})
	if err != nil {
		return []byte(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest
	}

	return b, http.StatusOK
}
