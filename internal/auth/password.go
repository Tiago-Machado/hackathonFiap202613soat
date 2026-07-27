package auth

import "golang.org/x/crypto/bcrypt"

const hashCost = bcrypt.DefaultCost

type BcryptHasher struct{}

func NewBcryptHasher() BcryptHasher {
	return BcryptHasher{}
}

func (BcryptHasher) Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), hashCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (BcryptHasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
