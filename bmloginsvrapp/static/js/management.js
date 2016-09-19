//	提交注册信息
$(document).ready(function(){
	//	search user
	var searchUserForm = $("#id-form-searchuser");
	if(null != searchUserForm)
	{
		searchUserForm.submit(function(event){
			var searchUserButton = $("#id-searchuser");
			event.preventDefault();
			var target = event.target;
			var action = $(target).attr("action");
			if(searchUserButton.hasClass("disabled"))
			{
				return;
			}
			//	set disable mode
			searchUserButton.addClass("disabled");
			$.post(action, $(target).serialize(), function(ret){
				searchUserButton.removeClass("disabled");
				var signUpHint = $("#id-signup-hint")
				var container = $("#id-result-container");
				
				if(null != ret.Result){
					if(0 != ret.Result){
						signUpHint.removeClass("hidden");
						$("#id-signup-hinttext").html(ret.Msg);
						
						if (!container.hasClass("hidden")) {
							container.addClass("hidden");
						}
					} else {
						//	insert dom
						var obj = $.parseJSON(ret.Msg);
						container.removeClass("hidden");
						if (!signUpHint.hasClass("hidden")) {
							signUpHint.addClass("hidden");
						}
						$("#text-uid").html(obj.Uid);
						$("#text-account").html(obj.Account);
						$("#text-password").html(obj.Password);
						$("#text-mail").html(obj.Mail);
						$("#link-adddonate").attr("href", "/management/adddonate?uid="+obj.Uid);
					}
				}
			}).error(function(e){
				searchUserButton.removeClass("disabled");
				$("#id-signup-hint").removeClass("hidden");
				$("#id-signup-hinttext").html("请求失败，请检查网络");
			});
		})
	}
	
	//	add donate
	var addDonateForm = $("#id-form-adddonate");
	if(null != addDonateForm)
	{
		addDonateForm.submit(function(event){
			var addDonateButton = $("#id-adddonate");
			event.preventDefault();
			var target = event.target;
			var action = $(target).attr("action");
			if(addDonateButton.hasClass("disabled"))
			{
				return;
			}
			//	set disable mode
			addDonateButton.addClass("disabled");
			$.post(action, $(target).serialize(), function(ret){
				addDonateButton.removeClass("disabled");
				var signUpHint = $("#id-signup-hint")
				var container = $("#id-result-container");
				
				if(null != ret.Result){
					if(0 != ret.Result){
						signUpHint.removeClass("hidden");
						$("#id-signup-hinttext").html(ret.Msg);
						
						if (!container.hasClass("hidden")) {
							container.addClass("hidden");
						}
					} else {
						//	insert dom
						if (!signUpHint.hasClass("hidden")) {
							signUpHint.addClass("hidden");
						}
						alert("添加成功");
					}
				}
			}).error(function(e){
				addDonateButton.removeClass("disabled");
				$("#id-signup-hint").removeClass("hidden");
				$("#id-signup-hinttext").html("请求失败，请检查网络");
			});
		})
	}
});