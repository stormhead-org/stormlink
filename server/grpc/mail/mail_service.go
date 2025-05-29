package mail

import (
	"stormlink/server/ent"
	"stormlink/server/grpc/mail/protobuf"
)

type MailService struct {
	protobuf.UnimplementedMailServiceServer
	client *ent.Client
}

func NewMailService(client *ent.Client) *MailService {
	return &MailService{client: client}
}
