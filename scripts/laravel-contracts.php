<?php
// Read-only contract probe: array cache/session and synthetic models only.
$root = $argv[1] ?? dirname(__DIR__, 2).'/SSO-Paradise-Supply-Chain';
putenv('APP_ENV=testing');
require $root.'/vendor/autoload.php';
$app = require $root.'/bootstrap/app.php';
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
config(['cache.default'=>'array', 'session.driver'=>'array', 'app.url'=>'http://localhost:3000', 'app.name'=>'Laravel', 'app.debug'=>false]);
$app->setLocale('fa');
$results = [];
foreach (['empty'=>[], 'invalid'=>['email'=>'invalid','password'=>12], 'extra'=>['email'=>'valid@example.com','password'=>'test','admin'=>true]] as $name=>$input) {
    $validator = validator($input, ['email'=>'required|email', 'password'=>'required|string']);
    if ($validator->fails()) {
        $exception = new Illuminate\Validation\ValidationException($validator);
        $results['login'][$name] = ['message'=>$exception->getMessage(), 'errors'=>$validator->errors()->toArray()];
    } else { $results['login'][$name] = null; }
}
$attributes = ['id'=>42,'name'=>'Test','email'=>'test@example.com','mobile'=>null,'email_verified_at'=>'2026-01-02 03:04:05','password'=>'secret','referral'=>null,'code'=>'hm-2000042','remember_token'=>'secret','created_at'=>'2026-01-02 03:04:05','updated_at'=>'2026-01-02 03:04:05','wallet_address'=>null];
$personalRequest = new App\Http\Requests\UpdatePersonalInfoRequest;
$validator = validator([], $personalRequest->rules());
$exception = new Illuminate\Validation\ValidationException($validator);
$results['personal_empty'] = ['message'=>$exception->getMessage(), 'errors'=>$validator->errors()->toArray()];
$user = new App\Models\User;
$user->setRawAttributes($attributes);
$results['user'] = $user->toArray();
$user->setRelation('personalInfo', (new App\Models\PersonalInfo)->forceFill(['is_verified'=>false, 'first_name'=>'First','last_name'=>'Last']));
$user->setRelation('media', new Illuminate\Database\Eloquent\Collection);
$request = Illuminate\Http\Request::create('http://localhost:3000/api/me', 'POST');
$results['resource'] = (new App\Http\Resources\UserResource($user))->toArray($request);
$user->personalInfo->is_verified = true;
$results['verified_resource'] = (new App\Http\Resources\UserResource($user))->toArray($request);
foreach (['invalid_client','invalid_grant','invalid_scope','unsupported_grant_type','access_denied','unsupported_response_type','invalid_request'] as $name) {
    $class = League\OAuth2\Server\Exception\OAuthServerException::class;
    $exception = match($name) {
        'invalid_client'=>$class::invalidClient(new GuzzleHttp\Psr7\ServerRequest('POST', '/oauth/token')),
        'invalid_grant'=>$class::invalidGrant(),
        'invalid_scope'=>$class::invalidScope('bad'),
        'unsupported_grant_type'=>$class::unsupportedGrantType(),
        'access_denied'=>$class::accessDenied(),
        'unsupported_response_type'=>$class::invalidRequest('response_type'),
        default=>$class::invalidRequest('grant_type'),
    };
    $response = $exception->generateHttpResponse(new GuzzleHttp\Psr7\Response);
    $results['oauth'][$name] = ['status'=>$response->getStatusCode(),'body'=>json_decode((string)$response->getBody(),true),'headers'=>$response->getHeaders()];
}
Illuminate\Support\Str::createRandomStringsUsing(fn ($length) => str_repeat('a', $length));
$web3 = new App\Http\Controllers\Auth\Web3AuthController;
$walletRequest = Illuminate\Http\Request::create('/web3/nonce', 'GET', ['address'=>'0x'.str_repeat('a',40)]);
$results['web3_nonce'] = $web3->getLoginNonce($walletRequest)->getData(true);
$method = new ReflectionMethod($web3, 'buildLinkMessage');
$results['web3_link_nonce'] = ['nonce'=>$method->invoke($web3, 42, '0x'.str_repeat('a',40))];
Illuminate\Support\Str::createRandomStringsNormally();
$results['passport_configuration'] = ['password'=>Laravel\Passport\Passport::$passwordGrantEnabled, 'device_code'=>Laravel\Passport\Passport::$deviceCodeGrantEnabled, 'legacy_clients'=>[]];
foreach (['authorization_code'=>[false,false,'https://client.example/callback'], 'password'=>[false,true,''], 'personal_access'=>[true,false,'']] as $kind=>[$personal,$password,$redirect]) {
    $client=(new App\Models\Passport\Client)->forceFill(['secret'=>'synthetic', 'personal_access_client'=>$personal,'password_client'=>$password,'redirect'=>$redirect]);
    $results['passport_configuration']['legacy_clients'][$kind]=$client->grant_types;
}
$results['routes'] = [];
foreach ($app['router']->getRoutes() as $route) {
    if (!str_starts_with($route->uri(), '_ignition')) {
        $results['routes'][] = ['uri'=>$route->uri(),'methods'=>$route->methods(),'middleware'=>$route->gatherMiddleware()];
    }
}
echo json_encode($results, JSON_PRETTY_PRINT|JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES|JSON_THROW_ON_ERROR);
