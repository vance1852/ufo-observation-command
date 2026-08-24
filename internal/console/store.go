package console

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ufo-observation-command/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	pool *db.Pool
}

func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Login(ctx context.Context, username, password string, ttl time.Duration) (Session, User, error) {
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	var user User
	var passwordHash string
	err := s.pool.QueryRow(ctx, `SELECT id,username,password_hash,real_name,phone,role,status FROM console_users WHERE username=$1`, strings.TrimSpace(username)).Scan(
		&user.ID, &user.Username, &passwordHash, &user.RealName, &user.Phone, &user.Role, &user.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, User{}, fmt.Errorf("用户名或密码错误")
	}
	if err != nil {
		return Session{}, User{}, fmt.Errorf("查询用户: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return Session{}, User{}, fmt.Errorf("用户名或密码错误")
	}
	if user.Status != 1 {
		return Session{}, User{}, fmt.Errorf("账号已被禁用")
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Session{}, User{}, fmt.Errorf("生成会话令牌: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = s.pool.Exec(ctx, `INSERT INTO console_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)`, tokenHash(token), user.ID, expiresAt)
	if err != nil {
		return Session{}, User{}, fmt.Errorf("创建登录会话: %w", err)
	}
	return Session{Token: token, ExpiresAt: expiresAt}, user, nil
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Store) UserBySession(ctx context.Context, token string) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `UPDATE console_sessions s SET last_seen_at=now()
		FROM console_users u
		WHERE s.token_hash=$1 AND s.user_id=u.id AND s.revoked_at IS NULL AND s.expires_at>now() AND u.status=1
		RETURNING u.id,u.username,u.real_name,u.phone,u.role,u.status`, tokenHash(strings.TrimSpace(token))).Scan(
		&user.ID, &user.Username, &user.RealName, &user.Phone, &user.Role, &user.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, fmt.Errorf("会话无效、已过期或已撤销")
	}
	return user, wrap("校验登录会话", err)
}

func (s *Store) RevokeSession(ctx context.Context, token string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash(strings.TrimSpace(token)))
	if err != nil {
		return fmt.Errorf("撤销登录会话: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("登录会话不存在或已撤销")
	}
	return nil
}

func (s *Store) Dashboard(ctx context.Context) (DashboardStats, error) {
	var stats DashboardStats
	err := s.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM console_acoustic_buoys WHERE deleted_at IS NULL AND status=1),
		(SELECT count(*) FROM console_array_operators WHERE deleted_at IS NULL AND status=1),
		(SELECT count(*) FROM console_command_orders WHERE status=0),
		(SELECT count(*) FROM console_command_orders WHERE status=2)`).Scan(
		&stats.AcousticBuoyCount, &stats.ArrayOperatorCount, &stats.PendingWorkOrders, &stats.CompletedWorkOrders,
	)
	return stats, wrap("统计首页数据", err)
}

func pageBounds(current, size int) (int, int, int) {
	if current < 1 {
		current = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	return current, size, (current - 1) * size
}

func wrap(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: 记录不存在", action)
	}
	return fmt.Errorf("%s: %w", action, err)
}

func (s *Store) WriteLog(ctx context.Context, username, operation, method, ip string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO console_logs(id,username,operation,method,ip) VALUES($1,$2,$3,$4,$5)`, uuid.NewString(), username, operation, method, ip)
	return wrap("记录操作日志", err)
}

func parseDate(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}
