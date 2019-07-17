package tpm2

import (
	"fmt"
)

func (t *tpmImpl) StartAuthSession(tpmKey, bind ResourceContext, sessionType SessionType, symmetric *SymDef,
	authHash AlgorithmId, authValue interface{}) (ResourceContext, error) {
	if tpmKey != nil {
		if err := t.checkResourceContextParam(tpmKey, "tpmKey"); err != nil {
			return nil, err
		}
	}
	if bind != nil {
		if err := t.checkResourceContextParam(bind, "bind"); err != nil {
			return nil, err
		}
	}
	if symmetric != nil {
		return nil, InvalidParamError{"no support for parameter / response encryption yet"}
	}
	digestSize, knownDigest := digestSizes[authHash]
	if !knownDigest {
		return nil, InvalidParamError{fmt.Sprintf("unsupported authHash value %v", authHash)}
	}

	var salt []byte
	var encryptedSalt EncryptedSecret

	if tpmKey != nil {
		object, isObject := tpmKey.(*objectContext)
		if !isObject {
			return nil, InvalidParamError{"tpmKey is not an object"}
		}

		var err error
		encryptedSalt, salt, err = cryptComputeEncryptedSalt(&object.public)
		if err != nil {
			return nil, fmt.Errorf("failed to compute encrypted salt: %v", err)
		}
	} else {
		tpmKey = &permanentContext{handle: HandleNull}
	}

	var authB []byte
	if bind != nil {
		switch a := authValue.(type) {
		case string:
			authB = []byte(a)
		case []byte:
			authB = a
		case nil:
		default:
			return nil, InvalidParamError{fmt.Sprintf("invalid auth value: %v", authValue)}
		}
	} else {
		bind, _ = t.WrapHandle(HandleNull)
	}

	nonceCaller := make([]byte, digestSize)
	if err := cryptComputeNonce(nonceCaller); err != nil {
		return nil, fmt.Errorf("cannot compute initial nonceCaller: %v", err)
	}

	var sessionHandle Handle
	var nonceTPM Nonce

	if err := t.RunCommand(CommandStartAuthSession, tpmKey, bind, Separator, Nonce(nonceCaller),
		encryptedSalt, sessionType, &SymDef{Algorithm: AlgorithmNull}, authHash, Separator,
		&sessionHandle, Separator, &nonceTPM); err != nil {
		return nil, err
	}

	sessionContext := &sessionContext{handle: sessionHandle,
		hashAlg:       authHash,
		boundResource: bind,
		nonceCaller:   Nonce(nonceCaller),
		nonceTPM:      nonceTPM}

	if tpmKey.Handle() != HandleNull || bind.Handle() != HandleNull {
		key := make([]byte, len(authB)+len(salt))
		copy(key, authB)
		copy(key[len(authB):], salt)

		sessionContext.sessionKey, _ =
			cryptKDFa(authHash, key, []byte("ATH"), []byte(nonceTPM), nonceCaller, digestSize*8)
	}

	t.addResourceContext(sessionContext)
	return sessionContext, nil
}
