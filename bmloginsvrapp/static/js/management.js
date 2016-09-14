//	提交注册信息
$(document).ready(function(){
	var searchUserButton = $("#id-searchuser");
	if(null != searchUserButton)
	{
		searchUserButton.submit(function(event){
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
				alert(ret);
				searchUserButton.removeClass("disabled");
				var signUpHint = $("#id-signup-hint")
				if(null != ret.Result){
					if(0 != ret.Result){
						signUpHint.removeClass("hidden");
						$("#id-signup-hinttext").html(ret.Msg);
					} else {
						//	insert dom
						var container = $("#id-result-container");
						container.empty();
						var obj = $.parseJSON(ret.Msg);
						
						alert(obj);
					}
				}
			}).error(function(e){
				searchUserButton.removeClass("disabled");
				$("#id-signup-hint").removeClass("hidden");
				$("#id-signup-hinttext").html("请求失败，请检查网络");
			});
		})
	}
});