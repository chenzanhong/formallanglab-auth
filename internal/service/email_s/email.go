// auth/internal/api/email/email.go
package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
	myErrors "github.com/chenzanhong/formallanglab-auth/internal/errors"
	"github.com/chenzanhong/formallanglab-auth/internal/repository"
	kafka_s "github.com/chenzanhong/formallanglab-auth/internal/service/kafka_s"
	"github.com/chenzanhong/formallanglab-auth/pkg/token"
	"github.com/chenzanhong/zlog"
	"gorm.io/gorm"
)

// ====== 频率限制与异步发送逻辑 ======
const (
	EmailTypeRegister = "register"
	EmailTypeResetPwd = "reset"

	// ⏱️ 请求间隔限制：1分钟内只能发1次（防刷）
	EmailSendRateLimitTTL = 1 * time.Minute
	// 验证码有效期保持1分钟
	VerificationTokenTTL = 1 * time.Minute
)

type EmailService interface {
	// 发送注册的验证码
	SendRegisterVerificationCode(ctx context.Context, email string) error

	// 发送找回密码的验证码
	SendResetPwdVerificationCode(ctx context.Context, email string) error
}

type EmailServiceImpl struct {
	emailRepo     repository.EmailRepository
	userRepo      repository.UserRepository
	kafkaProducer kafka_s.KafkaProducerService
}

func NewEmailService(emailRepo repository.EmailRepository, userRepo repository.UserRepository, kafkaProducer kafka_s.KafkaProducerService) EmailService {
	return &EmailServiceImpl{
		emailRepo:     emailRepo,
		userRepo:      userRepo,
		kafkaProducer: kafkaProducer,
	}
}

// ======================= 服务 =======================

// 服务：注册账号，发送验证码
func (s *EmailServiceImpl) SendRegisterVerificationCode(ctx context.Context, email string) error {
	// 限制频率，验证码有效期一分钟，不能重复发送
	if has, _ := s.emailRepo.HasRegisterVerificationToken(ctx, email); has {
		// zlog.Warnw("发送注册验证码", "detail", "操作太频繁，请稍后重试")
		return errors.New("操作太频繁，请稍后重试")
	}

	// 检查邮箱是否存在
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		zlog.Errorw("数据库查询失败", "error", err)
		return errors.New("系统异常")
	}
	if exists {
		return errors.New("邮箱已存在")
	}

	// 生成验证码
	verificationCode := token.GenerateRandomToken(6)
	zlog.Infow("发送注册验证码", "email", email, "code", verificationCode)

	// 保存token到Redis，过期时间1分钟
	if err := s.emailRepo.SaveRegisterVerificationToken(ctx, email, verificationCode); err != nil {
		return errors.New("保存验证码失败")
	}

	// 异步发送注册验证码邮件
	return s.sendRegisterEmail(email, verificationCode)
}

// 服务：处理重置密码请求，发送验证码
func (s *EmailServiceImpl) SendResetPwdVerificationCode(ctx context.Context, email string) error {
	// 限制频率，验证码有效期一分钟，不能重复发送
	if has, _ := s.emailRepo.HasResetPwdToken(ctx, email); has {
		zlog.Warnw("发送重置密码验证码", "detail", "操作太频繁，请稍后重试")
		return errors.New("操作太频繁，请稍后重试")
	}

	// 检查邮箱是否存在
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zlog.Warnw("重置密码请求", "detail", "用户未找到。")
			return myErrors.ErrUserNotFound
		} else {
			zlog.Errorw("重置密码请求", "detail", "数据库查询失败。")
			return myErrors.ErrInternal
		}
	}
	if !exists {
		return myErrors.ErrUserNotFound
	}

	// 生成 6 位数字 token
	token := token.GenerateRandomToken(6)
	zlog.Infow("生成找回密码 token", "email", email, "token", token)

	// 保存到 Redis，1 分钟过期
	if err := s.emailRepo.SaveResetPwdToken(ctx, token, email); err != nil {
		zlog.Errorw("保存找回密码 token 失败", "error", err)
		return myErrors.ErrInternal
	}

	// 发送重置密码邮件（异步），不处理错误，发送失败用户一分钟后重试
	s.sendResetPwdEmail(email, token)

	zlog.Infow("重置密码请求", "detail", "重置密码请求成功。")

	return nil
}

// ======================= 邮件生产者=======================

// 发送注册验证码邮件
func (s *EmailServiceImpl) sendRegisterEmail(email, code string) error {
	subject := "FormalLangLab 注册验证码"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f7f9fc;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
				<h1 style="margin: 0; font-size: 28px; font-weight: bold;">FormalLangLab</h1>
				<p style="margin: 10px 0 0 0; font-size: 16px; opacity: 0.9;">形式语言与自动机学习系统</p>
			</div>
			<div style="background-color: white; padding: 40px; border-radius: 0 0 10px 10px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);">
				<h2 style="color: #333; margin-bottom: 20px; font-size: 24px;">注册验证码</h2>
				<p style="color: #666; font-size: 16px; line-height: 1.6; margin-bottom: 30px;">感谢您注册 FormalLangLab 学习平台！请使用以下验证码完成注册：</p>
				<div style="background-color: #f8f9fa; border: 2px dashed #667eea; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0;">
					<span style="font-size: 32px; font-weight: bold; color: #667eea; letter-spacing: 6px; font-family: monospace;">%s</span>
				</div>
				<p style="color: #999; font-size: 14px; margin-bottom: 20px;">• 验证码有效期为 1 分钟</p>
				<p style="color: #999; font-size: 14px; margin-bottom: 20px;">• 请勿将验证码告知他人</p>
				<p style="color: #999; font-size: 14px;">如果您没有申请注册，请忽略此邮件。</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
				<p style="color: #999; font-size: 12px; text-align: center;">此邮件由 FormalLangLab 系统自动发送，请勿回复。</p>
			</div>
		</div>
	`, code)

	return s.sendEmailViaKafka(email, subject, "text/html", body)
}

// 发送重置密码的验证码邮件
func (s *EmailServiceImpl) sendResetPwdEmail(email, token string) error {
	subject := "FormalLangLab 重置密码"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f7f9fc;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
				<h1 style="margin: 0; font-size: 28px; font-weight: bold;">FormalLangLab</h1>
				<p style="margin: 10px 0 0 0; font-size: 16px; opacity: 0.9;">形式语言与自动机学习系统</p>
			</div>
			<div style="background-color: white; padding: 40px; border-radius: 0 0 10px 10px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);">
				<h2 style="color: #333; margin-bottom: 20px; font-size: 24px;">重置密码</h2>
				<p style="color: #666; font-size: 16px; line-height: 1.6; margin-bottom: 30px;">您正在请求重置 FormalLangLab 学习平台的密码，请使用以下验证码完成操作：</p>
				<div style="background-color: #f8f9fa; border: 2px dashed #667eea; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0;">
					<span style="font-size: 32px; font-weight: bold; color: #667eea; letter-spacing: 6px; font-family: monospace;">%s</span>
				</div>
				<p style="color: #999; font-size: 14px; margin-bottom: 20px;">• 验证码有效期为 1 分钟</p>
				<p style="color: #999; font-size: 14px; margin-bottom: 20px;">• 请勿将验证码告知他人</p>
				<p style="color: #999; font-size: 14px;">如果您没有请求重置密码，请忽略此邮件。</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
				<p style="color: #999; font-size: 12px; text-align: center;">此邮件由 FormalLangLab 系统自动发送，请勿回复。</p>
			</div>
		</div>
	`, token)

	return s.sendEmailViaKafka(email, subject, "text/html", body)
}

// SendEmailViaKafka 发送邮件事件到 Kafka
func (s *EmailServiceImpl) sendEmailViaKafka(email, subject, contextType, body string) error {
	event := &model.KafkaEmailEvent{
		To:          email,
		Subject:     subject,
		ContentType: contextType,
		Body:        body,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.kafkaProducer.SendEmailEvent(ctx, event); err != nil {
		return fmt.Errorf("发送邮件事件到 Kafka 失败: %w", err)
	}

	return nil
}
