//	菜单
$('li.dropdown').mouseover(function() {   
     $(this).addClass('open');    }).mouseout(function() {        $(this).removeClass('open');    }); 

function adjustFooter(){
	if ($(window).height() != $(document).height()) {
		$("#id_footer").removeClass("navbar-fixed-bottom");
	} else {
		if (!$("#id_footer").hasClass("navbar-fixed-bottom")) {
			$("#id_footer").addClass("navbar-fixed-bottom");
		}
	}
}

function adjustBodyMinHeight() {
	if($(window).height() == $(document).height()){
		var height = $(window).height() - $("#id_footer").height();
		$("body").css("min-height", height+"px");
	} else {
		$("body").css("min-height", "none");
	}
	
	//	set background-color
	/*$("body").css("background","#e5e5e5;-moz-linear-gradient(top,  #e5e5e5 0%, #ffffff 100%);-webkit-linear-gradient(top,  #e5e5e5 0%,#ffffff 100%);linear-gradient(to bottom,  #e5e5e5 0%,#ffffff 100%);");
	$("body").css("filter", "progid:DXImageTransform.Microsoft.gradient( startColorstr='#e5e5e5', endColorstr='#ffffff',GradientType=0 );");*/
}

$(document).ready(function(){
	adjustFooter();
	adjustBodyMinHeight();
})

$(window).resize(function(){
	adjustFooter();
	adjustBodyMinHeight();
})

$(document).resize(function(){
	adjustFooter();
	adjustBodyMinHeight();
});