package errorsx

import (
	"errors"
	"fmt"

	entruntime "entgo.io/ent/dialect/sql/sqlgraph"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPC мапит стандартные ошибки и ent-ошибки к gRPC status
func ToGRPC(err error, msg string) error {
    if err == nil { return nil }
    // ent runtime NotFound sentinel
    if isEntNotFound(err) {
        return status.Error(codes.NotFound, msg)
    }
    // Валидация/некорректные аргументы
    // тут можно добавить свои sentinels, например ErrUnauthorized, ErrForbidden и т.д.
    // по умолчанию — внутренняя ошибка с оборачиванием причины
    return status.Errorf(codes.Internal, "%s: %v", msg, err)
}

// FromGRPCCode помогает обернуть произвольную ошибку конкретным кодом
func FromGRPCCode(code codes.Code, msg string, cause error) error {
    if cause == nil { return status.Error(code, msg) }
    return status.Errorf(code, "%s: %v", msg, cause)
}

// GraphQLError упрощённый тип для нормализации GraphQL ошибок
type GraphQLError struct {
    Message string `json:"message"`
    Code    string `json:"code"`
}

// ToGraphQL нормализует ошибку к user-friendly сообщению и коду
// В более сложной версии можно мапить gRPC status → GraphQL extensions
func ToGraphQL(err error) *GraphQLError {
    if err == nil { return nil }
    st, ok := status.FromError(err)
    if ok {
        return &GraphQLError{Message: st.Message(), Code: st.Code().String()}
    }
    if isEntNotFound(err) {
        return &GraphQLError{Message: "not found", Code: codes.NotFound.String()}
    }
    return &GraphQLError{Message: fmt.Sprintf("internal error: %v", err), Code: codes.Internal.String()}
}

// isEntNotFound пытается распознать not found от ent без прямого импорта ent-пакетов моделей
func isEntNotFound(err error) bool {
    // sqlgraph NotFoundError используется ent при Only/OnlyX
    var nf *entruntime.NotFoundError
    if errors.As(err, &nf) { return true }
    // как fallback – по сообщению
    if status.Code(err) == codes.NotFound {
        return true
    }
    return false
}


