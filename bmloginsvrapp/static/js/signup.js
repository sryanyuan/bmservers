//	提交注册信息
$(document).ready(function(){
	var signUpForm = $("#id-form-signup");
	if(null != signUpForm)
	{
		signUpForm.submit(function(event){
			event.preventDefault();
			var target = event.target;
			var action = $(target).attr("action");
			if($("#id-signup-submit").hasClass("disabled"))
			{
				return;
			}
			//	set disable mode
			$("#id-signup-submit").addClass("disabled");
			$.post(action, $(target).serialize(), function(ret){
				$("#id-signup-submit").removeClass("disabled");
				var signUpHint = $("#id-signup-hint")
				if(null != ret.Result){
					if(0 != ret.Result){
						signUpHint.removeClass("hidden");
						$("#id-signup-hinttext").html(ret.Msg);

						//	refresh captcha?
						if(null != ret.CaptchaId &&
							ret.CaptchaId.length != 0){
							var signInCaptcha = $("#id-signup-captchaimg")
							signInCaptcha.attr("src", "/captcha/"+ret.CaptchaId+".png");
							$("#id-signup-captchaIdHolder").attr("value", ret.CaptchaId)
						}
					}else{
						location.href = ret.Msg;
					}
				}
			}).error(function(e){
				$("#id-signup-submit").removeClass("disabled");
				$("#id-signup-hint").removeClass("hidden");
				$("#id-signup-hinttext").html("请求失败，请检查网络");
			});
		})
	}

	var signInCaptcha = $("#id-signup-captchaimg")
	signInCaptcha.click(function(event){
		event.preventDefault();
		var refurl = $(this).attr("src")
		$(this).attr("src", refurl+"?reload="+(new Date()).getTime());
	});
});