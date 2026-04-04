package wallet

import "context"

type Service struct{}

func (s *Service) GetWalletByUserID(ctx context.Context, userID string) (string, string, error) {
	return "", "", nil
}
