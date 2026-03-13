package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

type BaseRepo[T, C any] interface {
	Delete(ctx context.Context, ids []int64) error
	UpdateByID(ctx context.Context, entity *T) error
	Create(ctx context.Context, entity *T) error
	Find(ctx context.Context, condition *C) (*T, error)
	FindAll(ctx context.Context, condition *C) ([]*T, error)
	ListPage(ctx context.Context, page, size int64, condition *C) ([]*T, int64, error)
	Count(ctx context.Context, condition *C) (int64, error)
	BatchCreate(ctx context.Context, list []*T) error
}

// baseRepo 是基于 gorm/gen 的通用仓储实现：
// T 为实体类型，C 为查询条件类型（通过 BuildDao 解析）。
type baseRepo[T, C any] struct {
	queryDAO func(ctx context.Context) gen.Dao
	idField  func(ctx context.Context) field.Int64
	id       func(entity *T) int64
	model    *T
}

// NewBaseRepo 创建通用仓储实例。
func NewBaseRepo[T, C any](
	queryDAO func(ctx context.Context) gen.Dao,
	idField func(ctx context.Context) field.Int64,
	id func(entity *T) int64,
	model *T,
) BaseRepo[T, C] {
	return baseRepo[T, C]{
		queryDAO: queryDAO,
		idField:  idField,
		id:       id,
		model:    model,
	}
}

func (b baseRepo[T, C]) Delete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	res, err := b.queryDAO(ctx).Where(b.idField(ctx).In(ids...)).Delete()
	if err != nil {
		return err
	}
	// 删除语句执行成功但未命中记录时，仅记录告警，不视为错误。
	if res.RowsAffected == 0 {
		log.Warnf("repo.Delete rows_affected=0 ids=%v", ids)
		return nil
	}
	return res.Error
}

func (b baseRepo[T, C]) UpdateByID(ctx context.Context, entity *T) error {
	if entity == nil {
		return errors.New("entity is nil")
	}
	id := b.id(entity)
	if id == 0 {
		return errors.New("entity id is required")
	}
	res, err := b.queryDAO(ctx).Where(b.idField(ctx).Eq(id)).Updates(entity)
	if err != nil {
		return err
	}
	// 更新语句执行成功但未命中记录时，仅记录告警，不视为错误。
	if res.RowsAffected == 0 {
		log.Warnf("repo.UpdateByID rows_affected=0 id=%d", id)
		return nil
	}
	return res.Error
}

func (b baseRepo[T, C]) Create(ctx context.Context, entity *T) error {
	if entity == nil {
		return errors.New("entity is nil")
	}
	return b.queryDAO(ctx).Create(entity)
}

func (b baseRepo[T, C]) Find(ctx context.Context, condition *C) (*T, error) {
	dao, err := BuildDao(b.queryDAO(ctx), b.model, condition)
	if err != nil {
		return nil, err
	}
	var result interface{}
	result, err = dao.First()
	if err != nil {
		return nil, err
	}
	item, ok := result.(*T)
	if !ok {
		return nil, fmt.Errorf("unexpected first type %T", result)
	}
	return item, nil
}

func (b baseRepo[T, C]) FindAll(ctx context.Context, condition *C) ([]*T, error) {
	dao, err := BuildDao(b.queryDAO(ctx), b.model, condition)
	if err != nil {
		return nil, err
	}
	var result interface{}
	result, err = dao.Find()
	if err != nil {
		return nil, err
	}
	list, ok := result.([]*T)
	if !ok {
		return nil, fmt.Errorf("unexpected find type %T", result)
	}
	return list, nil
}

func (b baseRepo[T, C]) ListPage(ctx context.Context, page, size int64, condition *C) ([]*T, int64, error) {
	dao, err := BuildDao(b.queryDAO(ctx), b.model, condition)
	if err != nil {
		return nil, 0, err
	}
	offset, limit := PageOffsetLimit(page, size)

	var result interface{}
	result, err = dao.Offset(int(offset)).Limit(int(limit)).Find()
	if err != nil {
		return nil, 0, err
	}
	list, ok := result.([]*T)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected find type %T", result)
	}
	var count int64
	count, err = dao.Offset(-1).Limit(-1).Count()
	if err != nil {
		return nil, 0, err
	}
	return list, count, nil
}

func (b baseRepo[T, C]) Count(ctx context.Context, condition *C) (int64, error) {
	dao, err := BuildDao(b.queryDAO(ctx), b.model, condition)
	if err != nil {
		return 0, err
	}
	return dao.Count()
}

func (b baseRepo[T, C]) BatchCreate(ctx context.Context, list []*T) error {
	if len(list) == 0 {
		return nil
	}
	batchSize := calcAutoBatchSize[T](list)
	return b.queryDAO(ctx).CreateInBatches(list, batchSize)
}

// PageOffsetLimit 统一处理分页参数，兜底 page=1、size=10。
func PageOffsetLimit(page, size int64) (offset, limit int64) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	offset = (page - 1) * size
	limit = size
	return
}
