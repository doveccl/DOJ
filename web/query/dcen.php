<?php
/**
	*This code is about an encryption algorithm created by Doveccl.
	*Email: i@doveccl.com
	*This code is under the license of GPL v2.0
	*/

static $enc_bin = array(1, 2, 4, 8, 16, 32, 64, 128);

/*two base chars of mapping*/
static $enc_a = 'a', $enc_b = 'b';

/*opnions for enc_seed */
static
	$enc_low = 1,
	$enc_upp = 2,
	$enc_num = 4,
	$enc_pun = 8,
	$enc_ord = 16,
	$enc_swa = 32,
	$enc_flo = 64,
	$enc_non = 128;

static $enc_clr = 15; // 00001111

/*
	enc_seed is an int-master way of encryption
	this algorithm provides adjustive ways to encrypt
	for example:
	
		a master number
	0   0   0   0   0   0   0   0
	non flo swa ord pun num upp low
	
	non: no use without flo
		just a flag while do flower process
	flo: flower the UINT8 use the seed itself
		a position is 1 means reserving
		this position of the code
	swa: swap two parts of UINT8
		and reverse the last part
		0100	   1101 -> 1011	   0100
		 |		  |
		first part last part
	ord: order for swa & flo
		0: flo first, swa second
		1: swa first, flo second
		ord is useful while only flo & swa are used
		or it will be a flag like non
	pun: add punctuation & operation symbol to mapping
	num: add numbers to mapping
	upp: add uppercase to mapping
	low: add lowcase to mapping (use automatically)
*/

$enc_map = array();
$oed_map = array();
$mod_cmp = 0;
$map_len = 0;
$times = 0;

function enc_init_map($seed)
{
	global $enc_map, $oed_map, $enc_low, $enc_upp, $enc_num, $enc_pun, $map_len;
	
	$enc_map = array();
	
	for ($i = ord('a'); $i <= ord('z'); $i++)
		$enc_map[] = $i;

	if ($seed & $enc_upp)
		for ($i = ord('A'); $i <= ord('Z'); $i++)
			$enc_map[] = $i;

	if ($seed & $enc_num)
		for ($i = ord('0'); $i <= ord('9'); $i++)
			$enc_map[] = $i;

	if ($seed & $enc_pun)
	{
		for ($i = 33; $i <= 47; $i++)
			$enc_map[] = $i;
		for ($i = 58; $i <= 64; $i++)
			$enc_map[] = $i;
		for ($i = 91; $i <= 96; $i++)
			$enc_map[] = $i;
		for ($i = 123; $i <= 126; $i++)
			$enc_map[] = $i;
	}

	shuffle($enc_map);

	$map_len = count($enc_map);
	
	$oed_map = $enc_map;
}

function enc_init_fill(&$s, $src_len)
{
	global $map_len;
	
	$last = $map_len - $src_len % $map_len;

	for ($i = 0; $i < $last; $i++)
		$s[] = rand(0, 255);
	$len = $src_len + $last;
	$s[$len - 1] = $last;

	return $len;
}

function __enc_swap(&$uchar)
{
	global $enc_bin;
	$b = &$enc_bin;
	
	$temp = 0;
	for ($p = 0; $p < 4; $p++)
		if ($b[$p] & $uchar)
			$temp |= $b[7 - $p];
	for ($p = 4; $p < 8; $p++)
		if ($b[$p] & $uchar)
			$temp |= $b[$p - 4];
	
	$uchar = $temp;
}

function enc_swap(&$s, $len)
{
	global $map_len, $oed_map;
	
	for ($i = 0; $i < $len; $i ++)
		__enc_swap($s[$i]);
	
	for ($i = 0; $i < $map_len; $i++)
		__enc_swap($oed_map[$i]);
}

function enc_flower(&$s, $len, $veins)
{
	global $map_len, $oed_map;
	
	for ($i = 0; $i < $len; $i ++)
		$s[$i] ^= $veins;
		
	for ($i = 0; $i < $map_len; $i++)
		$oed_map[$i] ^= $veins;
}

function enc_div($s, $iskey)
{
	global $enc_bin, $enc_clr, $times, $mod_cmp, $enc_map, $enc_a, $enc_b;
	$b = &$enc_bin;

	$o = 0;
	for ($p = 0; $p < 4; $p++)
		if ($s & $b[$p + 4])
			$o |= $b[$p];
	$s &= $enc_clr;
	
	if ($iskey)
	{
		$s += ord($enc_a);
		$o += ord($enc_b);
	}
	else
	{
		if ($s < $mod_cmp)
			$s = $enc_map[$s + 16 * rand(0, $times)];
		else $s = $enc_map[$s + 16 * rand(0, $times - 1)];

		if ($o < $mod_cmp)
			$o = $enc_map[$o + 16 * rand(0, $times)];
		else $o = $enc_map[$o + 16 * rand(0, $times - 1)];
	}
	
	return chr($s).chr($o);
}

function enc_mapping(&$src, $len)
{
	global $times, $mod_cmp, $enc_map, $oed_map, $map_len;
	
	$enc = "";
	
	$unit = $len / $map_len;
	$mod_cmp = $map_len % 16;
	$times = $map_len / 16;
	
	$cnt_src = 0;
	for ($i = 0; $i < $map_len; $i++)
	{
		$enc .= enc_div($oed_map[$i], true);
		for ($j = 0; $j < $unit; $j++)
			$enc .= enc_div($src[$cnt_src++], false);
	}
	
	return $enc;
}

function dec_get_map_len($seed)
{
	global $enc_low, $enc_upp, $enc_num, $enc_pun, $map_len;
	
	$map_len = 26;
	if ($enc_upp & $seed)
		$map_len += 26;
	if ($enc_num & $seed)
		$map_len += 10;
	if ($enc_pun & $seed)
		$map_len += 32;
}

function __dec_swap(&$uchar)
{
	global $enc_bin;
	$b = &$enc_bin;
	
	$temp = 0;
	for ($p = 0; $p < 4; $p++)
		if ($b[7 - $p] & $uchar)
			$temp |= $b[$p];
	for ($p = 4; $p < 8; $p++)
		if ($b[$p - 4] & $uchar)
			$temp |= $b[$p];
	$uchar = $temp;
}

function dec_map_recovery(&$s, $len, $seed)
{
	global $enc_map, $map_len, $enc_bin, $enc_flo, $enc_swa, $enc_ord, $enc_a, $enc_b;
	$enc_map = array();
	$b =&$enc_bin;
	
	$unit_len = $len / 2 / $map_len;
	
	for ($i = 0; $i < $len ; $i += 2)	
		if (! ($i / 2 % $unit_len))
		{
			$s[$i] -= ord($enc_a);
			$s[$i + 1] -= ord($enc_b);
			
			for ($p = 0; $p < 4; $p++)
				if ($s[$i + 1] & $b[$p])
					$s[$i] |= $b[$p + 4];

			if ($enc_flo & $seed && $enc_swa & $seed)
				if ($enc_ord & $seed)
				{
					$s[$i] ^= $seed;
					__dec_swap($s[$i]);
				}
				else
				{
					__dec_swap($s[$i]);
					$s[$i] ^= $seed;
				}
			else if ($enc_flo & $seed)
				$s[$i] ^= $seed;
			else if ($enc_swa & $seed)
				__dec_swap($s[$i]);		
			if ($s[$i] < 33 || $s[$i] > 126 || in_array($s[$i], $enc_map))
				return false;
			else
				$enc_map[] = $s[$i];
		}

	return true;
}

function dec_unmapping(&$s, $len)
{	
	global $map_len, $enc_map, $enc_bin;
	$b =&$enc_bin;
	
	$unit_len = $len / 2 / $map_len;
	$half_map = array();
	
	for ($i = 0; $i< $map_len; $i++)
		$half_map[$enc_map[$i]] = $i;
	
	for ($i = 0; $i < $len ; $i += 2)
		if ($i / 2 % $unit_len)
		{
			$s[$i] = $half_map[$s[$i]] % 16;
			$s[$i + 1] = $half_map[$s[$i + 1]] % 16;
			
			for ($p = 0; $p < 4; $p++)
				if ($s[$i + 1] & $b[$p])
					$s[$i] |= $b[$p + 4];
		}
	
	$dec_cnt = 0;
	for ($i = 0; $i < $len ; $i += 2)
		$s[$dec_cnt++] = $s[$i];
}

function dec_swap(&$s, $len)
{
	global $map_len;
	
	$unit_len = $len / $map_len;
	
	for ($i = 0; $i < $len; $i++)
		if ($i % $unit_len)
			__dec_swap($s[$i]);
}
	
function dec_flower(&$s, $len, $veins)
{
	global $map_len;
	
	$unit_len = $len / $map_len;
	
	for ($i = 0; $i < $len; $i++)
		if ($i % $unit_len)
			$s[$i] ^= $veins;
}

function dec_select(&$s, $len)
{
	global $map_len;
	
	$unit_len = $len / $map_len;
	$src_suffix = 0;
	
	for ($i = 0; $i < $len; $i++)
		if ($i % $unit_len)
			$s[$src_suffix++] = $s[$i];
	
	$src_len = $src_suffix - $s[$src_suffix - 1];
	
	$res = "";
	for ($i = 0; $i < $src_len; $i++)
		$res .= chr($s[$i]);
		
	return $res;
}

function dc_encrypt($src_code, $enc_seed)
{
	global $enc_flo, $enc_swa, $enc_ord;
	
	if (0 > $enc_seed && $enc_seed > 255)
		return "The seed should be in range 0~255......";
	
	if ($src_code == "")
		return "Sorry, empty string is not allowed !";
	
	$src_len = strlen($src_code);
	$src = array();
	
	for($i = 0; $i < $src_len; $i++)
		$src[] = ord($src_code[$i]);
	
	enc_init_map($enc_seed);
	$len = enc_init_fill($src, $src_len);

	if ($enc_flo & $enc_seed && $enc_swa & $enc_seed)
		if ($enc_ord & $enc_seed)
		{
			enc_swap($src, $len);
			enc_flower($src, $len, $enc_seed);
		}
		else
		{
			enc_flower($src, $len, $enc_seed);
			enc_swap($src, $len);
		}
	else if ($enc_flo & $enc_seed)
		enc_flower($src, $len, $enc_seed);
	else if ($enc_swa & $enc_seed)
		enc_swap($src, $len);

	return enc_mapping($src, $len);
}

function dc_decrypt($enc_code, $enc_seed)
{
	global $map_len, $enc_flo, $enc_swa, $enc_ord;
	dec_get_map_len($enc_seed);
	
	if (0 > $enc_seed && $enc_seed > 255)
		return "Wrong : The seed should be in range 0~255......";

	$len = strlen($enc_code);
	if ($len == 0)
		return "Wrong : Invalid encrypted code";
	if ($len % $map_len)
		return "Wrong : Your encrypted code does not match with seed......";
	
	$enc = array();
	for($i = 0; $i < $len; $i++)
		$enc[] = ord($enc_code[$i]);
	
	if(!dec_map_recovery($enc, $len, $enc_seed))
	    return "Wrong : Invalid encrypted code";
	dec_unmapping($enc, $len);
	
	$len /= 2;
	
	if ($enc_flo & $enc_seed && $enc_swa & $enc_seed)
		if ($enc_ord & $enc_seed)
		{
			dec_flower($enc, $len, $enc_seed);
			dec_swap($enc, $len);
		}
		else
		{
			dec_swap($enc, $len);
			dec_flower($enc, $len, $enc_seed);
		}
	else if ($enc_flo & $enc_seed)
		dec_flower($enc, $len, $enc_seed);
	else if ($enc_swa & $enc_seed)
		dec_swap($enc, $len);

	return dec_select($enc, $len);
}
