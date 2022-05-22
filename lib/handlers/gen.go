package handlers

const (
	BaseRespMessageOK = "ok"
)

type BaseResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

//NewErrorResp creates a new base response - should only be used for bad calls
func NewErrorResp(err error, msg string) *BaseResponse {
	return &BaseResponse{
		Ok:      false,
		Message: msg,
		Err:     err,
	}
}

//NewBaseResp creates a new base response - should only be used for good calls
func NewBaseResp() *BaseResponse {
	return &BaseResponse{
		Ok:      true,
		Message: BaseRespMessageOK,
	}
}
