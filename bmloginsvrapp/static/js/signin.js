//	提交登陆信息
$(document).ready(function(){
	var signInForm = $("#id-form-signin");
	if(null != signInForm){
		signInForm.submit(function(event){
			event.preventDefault();
			var target = event.target;
			var action = $(target).attr("action");
			if($("#id-signin-submit").hasClass("disabled")){
				return;
			}

			$("#id-signin-submit").addClass("disabled");
			$.post(action, $(target).serialize(), function(ret){
				$("#id-signin-submit").removeClass("disabled");
				var signInHint = $("#id-signin-hint")
				if(null != ret){
					if(0 != ret.Result){
						signInHint.removeClass("hidden");
						$("#id-signin-hinttext").html(ret.Msg);

						//	refresh captcha?
						if(null != ret.CaptchaId &&
							ret.CaptchaId.length != 0){
							var signInCaptcha = $("#id-signin-captchaimg")
							signInCaptcha.attr("src", "/captcha/"+ret.CaptchaId+".png");
							$("#id-signin-captchaIdHolder").attr("value", ret.CaptchaId)
						}
					}else{
						location.href = ret.Msg;
					}
				}
			}).error(function(e){
				$("#id-signin-submit").removeClass("disabled");
				$("#id-signin-hint").removeClass("hidden");
				$("#id-signin-hinttext").html("请求失败，请检查网络");
			});
		})
	}

	var signInCaptcha = $("#id-signin-captchaimg")
	signInCaptcha.click(function(event){
		event.preventDefault();
		var refurl = $(this).attr("src")
		$(this).attr("src", refurl+"?reload="+(new Date()).getTime());
	});
});