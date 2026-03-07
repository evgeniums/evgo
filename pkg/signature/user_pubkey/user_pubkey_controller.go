package user_pubkey

import (
	"errors"
	"net/http"

	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/signature"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/utils"
)

const (
	ErrorCodeDuplicateKey      string = "duplicate_pubkey"
	ErrorCodeActiveKeyNotFound string = "active_pubkey_not_found"
)

var ErrorDescriptions = map[string]string{
	ErrorCodeDuplicateKey:      "Duplicate key already loaded",
	ErrorCodeActiveKeyNotFound: "Active public key not found",
}

var ErrorHttpCodes = map[string]int{
	ErrorCodeActiveKeyNotFound: http.StatusUnauthorized,
}

type PubkeyController[T UserPubkeyI] interface {
	AddPubKey(ctx op_context.Context, userId string, key string, idIsLogin ...bool) (string, error)
	DeactivatePubKey(ctx op_context.Context, userId string, keyId string, idIsLogin ...bool) error
	FindActivePubKey(ctx op_context.Context, userId string, idIsLogin ...bool) (T, error)
}

type PubkeyControllerBase[T UserPubkeyI, U user.User] struct {
	crud             crud.CRUD
	objectBuilder    func() T
	userFinder       user.UserFinder[U]
	signatureManager signature.SignatureManager
}

func (p *PubkeyControllerBase[T, U]) SetUserFinder(userFinder user.UserFinder[U]) {
	p.userFinder = userFinder
}

func (p *PubkeyControllerBase[T, U]) CRUD() crud.CRUD {
	return p.crud
}

func (p *PubkeyControllerBase[T, U]) OpLog(ctx op_context.Context, op string, userId string, login string, keyId string, keyHash string) {
	oplog := NewOplog()
	oplog.SetOperation(op)
	oplog.SetLogin(login)
	oplog.SetUserId(userId)
	oplog.KeyId = keyId
	oplog.KeyHash = keyHash
	ctx.Oplog(oplog)
}

func (p *PubkeyControllerBase[T, U]) AddPubKey(ctx op_context.Context, userId string, key string, idIsLogin ...bool) (string, error) {

	// setup
	c := ctx.TraceInMethod("PubkeyController.AddPubKey")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// check key
	err = p.signatureManager.CheckPubKey(ctx, key)
	if err != nil {
		c.SetMessage("invalid key format")
		return "", err
	}

	// find user
	user, err := user.FindUser(p.userFinder, ctx, userId, idIsLogin...)
	if err != nil {
		return "", err
	}
	c.SetLoggerField("user", user.Display())

	// calculate hash
	hash := crypt_utils.H256B64([]byte(key))
	c.SetLoggerField("key_hash", hash)

	// run transaction
	doc := p.objectBuilder()
	err = op_context.ExecDbTransaction(ctx, func() error {

		// check if this key was already loaded
		filter := db.NewFilter()
		filter.AddField("public_key_owner", user.GetID())
		filter.AddField("public_key_hash", hash)
		exists, err := p.crud.Exists(ctx, filter, p.objectBuilder())
		if err != nil {
			c.SetMessage("failed to check if this key exists")
			return err
		}
		if exists {
			ctx.SetGenericErrorCode(ErrorCodeDuplicateKey)
			err = errors.New("duplicate pubkey")
			return err
		}

		// deactivate old key
		err = p.deactivateKey(ctx, c, user)
		if err != nil {
			return err
		}

		// create new key document
		doc.InitObject()
		doc.SetActive(true)
		doc.SetPubKey(key)
		doc.SetPubKeyHash(hash)
		doc.SetPubKeyOwner(user.GetID())
		err = p.crud.Create(ctx, doc)
		if err != nil {
			c.SetMessage("failed to create pubkey in database")
			return err
		}

		// done
		p.OpLog(ctx, "add_pubkey", user.GetID(), user.Login(), doc.GetID(), hash)
		return nil
	})
	if err != nil {
		return "", err
	}

	// done
	return doc.GetID(), nil
}

func (p *PubkeyControllerBase[T, U]) deactivateKey(ctx op_context.Context, c op_context.CallContext, user U, keyId ...string) error {

	doc := p.objectBuilder()
	fields := db.Fields{"public_key_owner": user.GetID(), "active": true}
	if len(keyId) != 0 && keyId[0] != "" {
		fields["id"] = keyId
	}
	found, err := p.crud.Read(ctx, fields, doc)
	if err != nil {
		c.SetMessage("failed to find previous key")
		return c.SetError(err)
	}
	if found {
		err = p.crud.Update(ctx, doc, db.Fields{"active": false})
		if err != nil {
			c.SetMessage("failed to deactivate previous key")
			return c.SetError(err)
		}
		p.OpLog(ctx, "deactivate_pubkey", user.GetID(), user.Login(), doc.GetID(), doc.PubKeyHash())
	}

	return nil
}

func (p *PubkeyControllerBase[T, U]) DeactivatePubKey(ctx op_context.Context, userId string, keyId string, idIsLogin ...bool) error {

	// setup
	c := ctx.TraceInMethod("PubkeyController.DeactivatePubKey", logger.Fields{"key_id": keyId})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := user.FindUser(p.userFinder, ctx, userId, idIsLogin...)
	if err != nil {
		return err
	}
	c.SetLoggerField("user", user.Display())

	// deactivate old key document
	err = p.deactivateKey(ctx, c, user, keyId)
	if err != nil {
		return err
	}

	// done
	return nil
}

func (p *PubkeyControllerBase[T, U]) FindActivePubKey(ctx op_context.Context, userId string, idIsLogin ...bool) (T, error) {

	// setup
	c := ctx.TraceInMethod("PubkeyController.FindActivePubKey")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	uId := userId
	useLogin := utils.OptionalArg(false, idIsLogin...)
	if useLogin {
		user, err := user.FindUser(p.userFinder, ctx, userId, idIsLogin...)
		if err != nil {
			return *new(T), err
		}
		c.SetLoggerField("user", user.Display())
		uId = user.GetID()
	}

	// find key
	doc := p.objectBuilder()
	fields := db.Fields{"public_key_owner": uId, "active": true}
	found, err := p.crud.Read(ctx, fields, doc)
	if err != nil {
		c.SetMessage("failed to find active public key")
		return *new(T), err
	}
	if !found {
		ctx.SetGenericErrorCode(ErrorCodeActiveKeyNotFound)
		err = errors.New("key not found")
		return *new(T), err
	}

	// done
	return doc, nil
}

func (p *PubkeyControllerBase[T, U]) ListPubKeys(ctx op_context.Context, filter *db.Filter) ([]T, int64, error) {

	// setup
	c := ctx.TraceInMethod("PubkeyController.ListPubKeys")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// read docs
	var docs []T
	count, err := p.crud.List(ctx, filter, &docs)
	if err != nil {
		c.SetMessage("failed to read documents from database")
		return nil, 0, err
	}

	// done
	return docs, count, nil
}

func (p *PubkeyControllerBase[T, U]) AttachToErrorManager(errManager generic_error.ErrorManager) {
	errManager.AddErrorDescriptions(ErrorDescriptions)
	errManager.AddErrorProtocolCodes(ErrorHttpCodes)
}

func NewPubkeyController[T UserPubkeyI, U user.User](objectBuilder func() T,
	signatureManager signature.SignatureManager,
	userFinder user.UserFinder[U],
	cruds ...crud.CRUD) *PubkeyControllerBase[T, U] {

	p := &PubkeyControllerBase[T, U]{objectBuilder: objectBuilder, signatureManager: signatureManager, userFinder: userFinder}

	if len(cruds) == 0 {
		p.crud = &crud.DbCRUD{}
	} else {
		p.crud = cruds[0]
	}

	return p
}

func ListPubkeys[T UserPubkeyI](crud crud.CRUD, ctx op_context.Context, filter *db.Filter, keyModel UserPubkeyI, userModel user.User, destModel T, queryName string) ([]T, int64, error) {

	// setup
	c := ctx.TraceInMethod(queryName)
	defer ctx.TraceOutMethod()

	// construct join query
	queryBuilder := func() (db.JoinQuery, error) {
		return ctx.Db().Joiner().
			Join(keyModel, "public_key_owner").On(userModel, "id").
			Destination(destModel)
	}

	// invoke join
	var pubkeys []T
	count, err := crud.Join(ctx, db.NewJoin(queryBuilder, queryName), filter, &pubkeys)
	if err != nil {
		c.SetMessage("failed to list public keys")
		return nil, 0, c.SetError(err)
	}

	// done
	return pubkeys, count, nil
}
