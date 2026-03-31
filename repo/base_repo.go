package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// BaseRepo 定义通用仓储能力。
type BaseRepo[T any] interface {
	Create(ctx context.Context, entity *T) error
	BatchCreate(ctx context.Context, list []*T) error
	Delete(ctx context.Context, opts ...QueryOption) error
	DeleteById(ctx context.Context, id int64) error
	DeleteByIds(ctx context.Context, ids []int64) error
	Update(ctx context.Context, entity *T, opts ...QueryOption) error
	UpdateById(ctx context.Context, entity *T) error
	Find(ctx context.Context, opts ...QueryOption) (*T, error)
	FindById(ctx context.Context, id int64) (*T, error)
	List(ctx context.Context, opts ...QueryOption) ([]*T, error)
	ListByIds(ctx context.Context, ids []int64) ([]*T, error)
	Page(ctx context.Context, page, size int64, opts ...QueryOption) ([]*T, int64, error)
	Count(ctx context.Context, opts ...QueryOption) (int64, error)
}

// baseRepo 是基于 gorm/gen 的通用仓储实现。
type baseRepo[T any] struct {
	queryDAO func(ctx context.Context) gen.Dao
	idField  func(ctx context.Context) field.Int64
	id       func(entity *T) int64
}

// NewBaseRepo 创建通用仓储实例。
func NewBaseRepo[T any](
	queryDAO func(ctx context.Context) gen.Dao,
	idField func(ctx context.Context) field.Int64,
	id func(entity *T) int64,
) BaseRepo[T] {
	return baseRepo[T]{
		queryDAO: queryDAO,
		idField:  idField,
		id:       id,
	}
}

// Create 创建单条记录。
func (b baseRepo[T]) Create(ctx context.Context, entity *T) error {
	if entity == nil {
		return errors.New("entity is nil")
	}
	return b.queryDAO(ctx).Create(entity)
}

// BatchCreate 批量创建记录。
func (b baseRepo[T]) BatchCreate(ctx context.Context, list []*T) error {
	if len(list) == 0 {
		return nil
	}
	batchSize := calcAutoBatchSize[T](list)
	return b.queryDAO(ctx).CreateInBatches(list, batchSize)
}

// Delete 按查询条件删除记录。
// 为避免误删全表，必须显式传入至少一个查询选项。
func (b baseRepo[T]) Delete(ctx context.Context, opts ...QueryOption) error {
	if err := validateRequiredQueryOptions(opts...); err != nil {
		return err
	}
	dao := ApplyQueryOptions(b.queryDAO(ctx), opts...)
	res, err := dao.Delete()
	if err != nil {
		return err
	}
	return res.Error
}

// DeleteById 按主键删除单条记录。
func (b baseRepo[T]) DeleteById(ctx context.Context, id int64) error {
	if id == 0 {
		return nil
	}
	res, err := b.queryDAO(ctx).Where(b.idField(ctx).Eq(id)).Delete()
	if err != nil {
		return err
	}
	// 删除语句执行成功但未命中记录时，仅记录告警，不视为错误。
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}

// DeleteByIds 按主键批量删除记录。
func (b baseRepo[T]) DeleteByIds(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	res, err := b.queryDAO(ctx).Where(b.idField(ctx).In(ids...)).Delete()
	if err != nil {
		return err
	}
	// 删除语句执行成功但未命中记录时，仅记录告警，不视为错误。
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}

// Update 按查询条件批量更新记录。
// 为避免误更新全表，必须显式传入至少一个查询选项。
func (b baseRepo[T]) Update(ctx context.Context, entity *T, opts ...QueryOption) error {
	if entity == nil {
		return errors.New("entity is nil")
	}
	if err := validateRequiredQueryOptions(opts...); err != nil {
		return err
	}
	dao := ApplyQueryOptions(b.queryDAO(ctx), opts...)
	res, err := dao.Updates(entity)
	if err != nil {
		return err
	}
	// 更新语句执行成功但未命中记录时，仅记录告警，不视为错误。
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}

// UpdateById 按主键更新记录。
func (b baseRepo[T]) UpdateById(ctx context.Context, entity *T) error {
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
		return nil
	}
	return res.Error
}

// Find 根据条件查询单条记录。
func (b baseRepo[T]) Find(ctx context.Context, opts ...QueryOption) (*T, error) {
	if err := validateRequiredQueryOptions(opts...); err != nil {
		return nil, err
	}
	dao := ApplyQueryOptions(b.queryDAO(ctx), opts...)
	result, err := dao.First()
	if err != nil {
		return nil, normalizeNotFoundError(err)
	}
	item, ok := result.(*T)
	if !ok {
		return nil, fmt.Errorf("unexpected first type %T", result)
	}
	return item, nil
}

// FindById 根据ID查询单条记录。
func (b baseRepo[T]) FindById(ctx context.Context, id int64) (*T, error) {
	if id == 0 {
		return nil, errors.New("id is required")
	}
	result, err := b.queryDAO(ctx).Where(b.idField(ctx).Eq(id)).First()
	if err != nil {
		return nil, normalizeNotFoundError(err)
	}
	item, ok := result.(*T)
	if !ok {
		return nil, fmt.Errorf("unexpected first type %T", result)
	}
	return item, nil
}

// List 查询列表。
func (b baseRepo[T]) List(ctx context.Context, opts ...QueryOption) ([]*T, error) {
	dao := ApplyQueryOptions(b.queryDAO(ctx), opts...)
	result, err := dao.Find()
	if err != nil {
		return nil, err
	}
	list, ok := result.([]*T)
	if !ok {
		return nil, fmt.Errorf("unexpected find type %T", result)
	}
	return list, nil
}

// ListByIds 根据ID列表查询列表
func (b baseRepo[T]) ListByIds(ctx context.Context, ids []int64) ([]*T, error) {
	if len(ids) == 0 {
		return []*T{}, nil
	}
	result, err := b.queryDAO(ctx).Where(b.idField(ctx).In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	list, ok := result.([]*T)
	if !ok {
		return nil, fmt.Errorf("unexpected find type %T", result)
	}
	return list, nil
}

// Page 查询分页列表。
func (b baseRepo[T]) Page(ctx context.Context, page, size int64, opts ...QueryOption) ([]*T, int64, error) {
	dao := ApplyQueryOptions(b.queryDAO(ctx), opts...)
	offset, limit := PageOffsetLimit(page, size)

	result, err := dao.Offset(int(offset)).Limit(int(limit)).Find()
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

// Count 查询条数
func (b baseRepo[T]) Count(ctx context.Context, opts ...QueryOption) (int64, error) {
	dao := ApplyQueryOptions(b.queryDAO(ctx), opts...)
	return dao.Count()
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

// validateRequiredQueryOptions 校验必须存在至少一个有效查询选项。
func validateRequiredQueryOptions(opts ...QueryOption) error {
	if len(opts) == 0 {
		return errors.New("opts is required")
	}
	for _, opt := range opts {
		if opt != nil {
			return nil
		}
	}
	return errors.New("opts is required")
}

// normalizeNotFoundError 统一将“记录不存在”归一到 gorm.ErrRecordNotFound，兼容底层返回字符串错误的场景。
func normalizeNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gorm.ErrRecordNotFound
	}
	if strings.EqualFold(strings.TrimSpace(err.Error()), gorm.ErrRecordNotFound.Error()) {
		return gorm.ErrRecordNotFound
	}
	return err
}
