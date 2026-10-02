"use client";
import { AccountMenu, SessionControl } from "./session-controls";
// Markup and artwork preserved from the Laravel layout; interactions are React-owned.
export function LegacyHeader({
  openNav,
  closeNav,
  setTheme,
}: {
  openNav: () => void;
  closeNav: () => void;
  setTheme: (dark: boolean) => void;
}) {
  return (
    <header>
      <nav className={" w-full m-0 p-0 fixed top-0 z-[1000]  lg:hidden"}>
        <div
          className={
            "w-full  items-center flex justify-between p-4 py-0 bg-white  dark:bg-[#0F0F0E] "
          }
        >
          <div
            className={
              "flex justify-between items-center w-full fixed right-0 top-0 lg:relative  bg-white dark:bg-[#0F0F0E] lg:bg-transparent lg:p-0 px-5 py-[10px]"
            }
          >
            <div className={"flex items-center justify-between gap-5 w-full"}>
              <div
                className={"w-max  p-2 rounded-full cursor-pointer flex"}
                onClick={openNav}
                role="button"
                tabIndex={0}
                aria-label="باز کردن فهرست"
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    openNav();
                  }
                }}
              >
                <svg
                  className={"lg:hidden dark:fill-white"}
                  width={"20"}
                  height={"20"}
                  viewBox={"0 0 30 22"}
                  fill={"none"}
                  xmlns={"http://www.w3.org/2000/svg"}
                >
                  <path
                    className={"dark:fill-white"}
                    fillRule={"evenodd"}
                    clipRule={"evenodd"}
                    d={
                      "M0 1.375C0 0.615608 0.6296 0 1.40625 0H28.5938C29.3704 0 30 0.615608 30 1.375C30 2.13439 29.3704 2.75 28.5938 2.75H1.40625C0.6296 2.75 0 2.13439 0 1.375ZM0 11C0 10.2406 0.6296 9.625 1.40625 9.625H28.5938C29.3704 9.625 30 10.2406 30 11C30 11.7594 29.3704 12.375 28.5938 12.375H1.40625C0.6296 12.375 0 11.7594 0 11ZM0 20.625C0 19.8656 0.6296 19.25 1.40625 19.25H28.5938C29.3704 19.25 30 19.8656 30 20.625C30 21.3844 29.3704 22 28.5938 22H1.40625C0.6296 22 0 21.3844 0 20.625Z"
                    }
                    fill={"#0713EF"}
                  ></path>
                </svg>
              </div>

              <div className={"flex items-center gap-2"}>
                <a
                  aria-label={"logo"}
                  className={"w-[35px] h-[35px]"}
                  href={"/home"}
                >
                  <img
                    className={"w-full h-full"}
                    src={"/images/logo/accounts.png"}
                    alt={"تونل زمان"}
                  />
                </a>
                <div className={"flex flex-col justify-between "}>
                  <span
                    className={"text-[#1A1A18] dark:text-[#FFFFFF] text-[13px]"}
                  >
                    {"تونل زمان"}
                  </span>
                  <span className={"text-[#939393] text-xs"}>
                    {"ورود / ثبت نام مرکزی"}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </nav>

      <div id={"main-nav"} className={"sidenav  z-[5000]"}>
        <div
          id={"open00"}
          dir={"ltr"}
          className={
            "hidden bg-white  dark:bg-[#0F0F0E]  p-4  relative  pr-0 h-full overflow-y-scroll scrollbar  z-[5000]"
          }
        >
          <nav
            dir={"rtl"}
            className={"w-full   space-y-6  relative lg:overflow-hidden"}
          >
            <div
              className={
                "  lg:h-full lg:pb-[220px] overflow-y-scroll scrollbar  space-y-1 relative"
              }
            >
              <div
                className={
                  "space-y-6   sticky top-0 mt-[-10px] pb-3 border-b-2 border-[#00000017] dark:border-[#3F3F3F] bg-white dark:bg-[#0F0F0E] z-50"
                }
              >
                <div
                  className={
                    " gap-5  my-2 w-full  items-center flex justify-between pr-5  bg-white  dark:bg-[#0F0F0E] "
                  }
                >
                  <div className={"flex items-center gap-2"}>
                    <a
                      aria-label={"logo"}
                      className={"w-[38px] h-[38px]"}
                      href={"/home"}
                    >
                      <img
                        className={"w-full h-full"}
                        src={"/images/logo/accounts.png"}
                        alt={"تونل زمان"}
                      />
                    </a>
                    <div className={"flex flex-col justify-between "}>
                      <span
                        className={"text-[#1A1A18] dark:text-[#FFFFFF] text-lg"}
                      >
                        {"تونل زمان"}
                      </span>
                      <span className={"text-[#939393] text-sm"}>
                        {"ورود / ثبت نام مرکزی"}
                      </span>
                    </div>
                  </div>

                  <div>
                    <div
                      id={"close-nav-btn"}
                      className={
                        "bg-slate-200 dark:bg-[#090909] w-10 h-10 p-3 rounded-full cursor-pointer flex items-center justify-center  "
                      }
                      onClick={closeNav}
                      role="button"
                      tabIndex={0}
                      aria-label="بستن فهرست"
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          closeNav();
                        }
                      }}
                    >
                      <img
                        src={
                          "https://3d.irpsc.com/home-page/images/aroowww.svg"
                        }
                        alt={""}
                        className={"w-[60%] "}
                      />
                    </div>
                  </div>
                </div>
              </div>
              <AccountMenu />
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <svg
                  width={"22"}
                  height={"22"}
                  viewBox={"0 0 22 22"}
                  fill={"none"}
                  xmlns={"http://www.w3.org/2000/svg"}
                >
                  <path
                    d={
                      "M6.86523 11H6.11523H6.86523ZM14.8652 11H15.6152H14.8652ZM10.8652 15V15.75V15ZM12.9346 20.7856L13.0891 21.5195L12.9346 20.7856ZM8.79583 20.7856L8.6414 21.5195L8.79583 20.7856ZM1.60384 7.22119L0.909524 6.93761H0.909524L1.60384 7.22119ZM1.07962 13.0694L0.345688 13.2238L1.07962 13.0694ZM8.79583 1.21438L8.6414 0.480454L8.79583 1.21438ZM12.9346 1.21438L13.0891 0.480455L12.9346 1.21438ZM20.0461 7.25728L20.2638 7.97499L20.0461 7.25728ZM1.6832 7.25692L1.9009 6.53921L1.6832 7.25692ZM19.4333 7.50731C19.8727 8.58399 20.1152 9.76274 20.1152 11H21.6152C21.6152 9.56509 21.3337 8.19407 20.8221 6.94055L19.4333 7.50731ZM20.1152 11C20.1152 11.6573 20.0468 12.2978 19.9169 12.915L21.3848 13.2238C21.5359 12.5057 21.6152 11.7617 21.6152 11H20.1152ZM19.9169 12.915C19.1655 16.4862 16.3514 19.3002 12.7802 20.0517L13.0891 21.5195C17.2422 20.6456 20.5109 17.377 21.3848 13.2238L19.9169 12.915ZM12.7802 20.0517C12.163 20.1816 11.5225 20.25 10.8652 20.25V21.75C11.6269 21.75 12.3709 21.6707 13.0891 21.5195L12.7802 20.0517ZM10.8652 20.25C10.208 20.25 9.56748 20.1816 8.95026 20.0517L8.6414 21.5195C9.35957 21.6707 10.1035 21.75 10.8652 21.75V20.25ZM1.61523 11C1.61523 9.76178 1.85813 8.58215 2.29816 7.50478L0.909524 6.93761C0.397214 8.19193 0.115235 9.56397 0.115235 11H1.61523ZM8.95026 20.0517C5.37906 19.3002 2.565 16.4862 1.81354 12.915L0.345688 13.2238C1.21959 17.377 4.48828 20.6456 8.6414 21.5195L8.95026 20.0517ZM1.81354 12.915C1.68367 12.2978 1.61523 11.6573 1.61523 11H0.115235C0.115235 11.7617 0.19457 12.5057 0.345688 13.2238L1.81354 12.915ZM2.29816 7.50478C3.44348 4.70064 5.92609 2.58466 8.95026 1.94831L8.6414 0.480454C5.1226 1.22088 2.2401 3.6799 0.909524 6.93761L2.29816 7.50478ZM8.95026 1.94831C9.56748 1.81844 10.208 1.75 10.8652 1.75V0.25C10.1035 0.25 9.35957 0.329336 8.6414 0.480454L8.95026 1.94831ZM10.8652 1.75C11.5225 1.75 12.163 1.81844 12.7802 1.94831L13.0891 0.480455C12.3709 0.329336 11.6269 0.25 10.8652 0.25V1.75ZM12.7802 1.94831C15.8053 2.58485 18.2885 4.70192 19.4333 7.50731L20.8221 6.94055C19.4921 3.68138 16.6089 1.22111 13.0891 0.480455L12.7802 1.94831ZM12.2201 1.44237C12.4961 2.30712 13.5174 5.63305 13.9361 8.68885L15.4222 8.48526C14.9849 5.29348 13.9292 1.86407 13.6492 0.986399L12.2201 1.44237ZM13.9361 8.68885C14.0483 9.5078 14.1152 10.2943 14.1152 11H15.6152C15.6152 10.2056 15.5403 9.34758 15.4222 8.48526L13.9361 8.68885ZM19.8284 6.53956C18.7309 6.87243 16.6696 7.45704 14.543 7.84951L14.8153 9.3246C17.0208 8.91754 19.1426 8.31505 20.2638 7.97499L19.8284 6.53956ZM14.543 7.84951C13.253 8.0876 11.9649 8.25 10.8652 8.25V9.75C12.0945 9.75 13.4831 9.57046 14.8153 9.3246L14.543 7.84951ZM14.1152 11C14.1152 12.0455 13.9684 13.2621 13.7489 14.4899L15.2255 14.7539C15.4527 13.4833 15.6152 12.1698 15.6152 11H14.1152ZM13.7489 14.4899C13.2677 17.1811 12.4593 19.808 12.2201 20.5576L13.6492 21.0136C13.8946 20.2443 14.7268 17.5432 15.2255 14.7539L13.7489 14.4899ZM20.4229 12.3549C19.6733 12.5941 17.0464 13.4025 14.3552 13.8836L14.6192 15.3602C17.4084 14.8615 20.1096 14.0294 20.8788 13.7839L20.4229 12.3549ZM14.3552 13.8836C13.1274 14.1032 11.9107 14.25 10.8652 14.25V15.75C12.0351 15.75 13.3485 15.5874 14.6192 15.3602L14.3552 13.8836ZM10.8652 14.25C9.81975 14.25 8.6031 14.1032 7.3753 13.8836L7.11129 15.3602C8.38194 15.5874 9.69541 15.75 10.8652 15.75V14.25ZM7.3753 13.8836C4.6841 13.4025 2.0572 12.5941 1.3076 12.3549L0.851632 13.7839C1.62092 14.0294 4.32207 14.8615 7.11129 15.3602L7.3753 13.8836ZM6.11523 11C6.11523 12.1698 6.27782 13.4833 6.50501 14.7539L7.98159 14.4899C7.76206 13.2621 7.61523 12.0455 7.61523 11H6.11523ZM6.50501 14.7539C7.00371 17.5432 7.83586 20.2443 8.08132 21.0136L9.51034 20.5576C9.27116 19.808 8.46277 17.1811 7.98159 14.4899L6.50501 14.7539ZM8.08132 0.986398C7.80127 1.86407 6.74553 5.29348 6.30828 8.48526L7.7944 8.68885C8.21302 5.63305 9.23442 2.30712 9.51034 1.44237L8.08132 0.986398ZM6.30828 8.48526C6.19015 9.34758 6.11523 10.2056 6.11523 11H7.61523C7.61523 10.2943 7.68221 9.5078 7.7944 8.68885L6.30828 8.48526ZM10.8652 8.25C9.76558 8.25 8.47748 8.0876 7.18746 7.84951L6.91522 9.3246C8.24735 9.57046 9.63601 9.75 10.8652 9.75V8.25ZM7.18746 7.84951C5.06013 7.4569 2.9981 6.87202 1.9009 6.53921L1.4655 7.97463C2.58638 8.31462 4.70884 8.9174 6.91522 9.3246L7.18746 7.84951ZM19.7821 6.55831C19.7954 6.55142 19.8113 6.54476 19.8284 6.53956L20.2638 7.97499C20.3377 7.95255 20.4074 7.92382 20.4734 7.88955L19.7821 6.55831ZM1.21255 7.86103C1.29027 7.90856 1.37463 7.94706 1.4655 7.97463L1.9009 6.53921C1.93557 6.54972 1.96753 6.56448 1.99514 6.58136L1.21255 7.86103Z"
                    }
                    fill={"#888888"}
                  ></path>
                </svg>
                <a href={"./contactus.html"} className={"text-[#868B90]"}>
                  {" تماس با ما"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <svg
                  width={"22"}
                  height={"22"}
                  viewBox={"0 0 22 22"}
                  fill={"none"}
                  xmlns={"http://www.w3.org/2000/svg"}
                >
                  <path
                    d={
                      "M6.86523 11H6.11523H6.86523ZM14.8652 11H15.6152H14.8652ZM10.8652 15V15.75V15ZM12.9346 20.7856L13.0891 21.5195L12.9346 20.7856ZM8.79583 20.7856L8.6414 21.5195L8.79583 20.7856ZM1.60384 7.22119L0.909524 6.93761H0.909524L1.60384 7.22119ZM1.07962 13.0694L0.345688 13.2238L1.07962 13.0694ZM8.79583 1.21438L8.6414 0.480454L8.79583 1.21438ZM12.9346 1.21438L13.0891 0.480455L12.9346 1.21438ZM20.0461 7.25728L20.2638 7.97499L20.0461 7.25728ZM1.6832 7.25692L1.9009 6.53921L1.6832 7.25692ZM19.4333 7.50731C19.8727 8.58399 20.1152 9.76274 20.1152 11H21.6152C21.6152 9.56509 21.3337 8.19407 20.8221 6.94055L19.4333 7.50731ZM20.1152 11C20.1152 11.6573 20.0468 12.2978 19.9169 12.915L21.3848 13.2238C21.5359 12.5057 21.6152 11.7617 21.6152 11H20.1152ZM19.9169 12.915C19.1655 16.4862 16.3514 19.3002 12.7802 20.0517L13.0891 21.5195C17.2422 20.6456 20.5109 17.377 21.3848 13.2238L19.9169 12.915ZM12.7802 20.0517C12.163 20.1816 11.5225 20.25 10.8652 20.25V21.75C11.6269 21.75 12.3709 21.6707 13.0891 21.5195L12.7802 20.0517ZM10.8652 20.25C10.208 20.25 9.56748 20.1816 8.95026 20.0517L8.6414 21.5195C9.35957 21.6707 10.1035 21.75 10.8652 21.75V20.25ZM1.61523 11C1.61523 9.76178 1.85813 8.58215 2.29816 7.50478L0.909524 6.93761C0.397214 8.19193 0.115235 9.56397 0.115235 11H1.61523ZM8.95026 20.0517C5.37906 19.3002 2.565 16.4862 1.81354 12.915L0.345688 13.2238C1.21959 17.377 4.48828 20.6456 8.6414 21.5195L8.95026 20.0517ZM1.81354 12.915C1.68367 12.2978 1.61523 11.6573 1.61523 11H0.115235C0.115235 11.7617 0.19457 12.5057 0.345688 13.2238L1.81354 12.915ZM2.29816 7.50478C3.44348 4.70064 5.92609 2.58466 8.95026 1.94831L8.6414 0.480454C5.1226 1.22088 2.2401 3.6799 0.909524 6.93761L2.29816 7.50478ZM8.95026 1.94831C9.56748 1.81844 10.208 1.75 10.8652 1.75V0.25C10.1035 0.25 9.35957 0.329336 8.6414 0.480454L8.95026 1.94831ZM10.8652 1.75C11.5225 1.75 12.163 1.81844 12.7802 1.94831L13.0891 0.480455C12.3709 0.329336 11.6269 0.25 10.8652 0.25V1.75ZM12.7802 1.94831C15.8053 2.58485 18.2885 4.70192 19.4333 7.50731L20.8221 6.94055C19.4921 3.68138 16.6089 1.22111 13.0891 0.480455L12.7802 1.94831ZM12.2201 1.44237C12.4961 2.30712 13.5174 5.63305 13.9361 8.68885L15.4222 8.48526C14.9849 5.29348 13.9292 1.86407 13.6492 0.986399L12.2201 1.44237ZM13.9361 8.68885C14.0483 9.5078 14.1152 10.2943 14.1152 11H15.6152C15.6152 10.2056 15.5403 9.34758 15.4222 8.48526L13.9361 8.68885ZM19.8284 6.53956C18.7309 6.87243 16.6696 7.45704 14.543 7.84951L14.8153 9.3246C17.0208 8.91754 19.1426 8.31505 20.2638 7.97499L19.8284 6.53956ZM14.543 7.84951C13.253 8.0876 11.9649 8.25 10.8652 8.25V9.75C12.0945 9.75 13.4831 9.57046 14.8153 9.3246L14.543 7.84951ZM14.1152 11C14.1152 12.0455 13.9684 13.2621 13.7489 14.4899L15.2255 14.7539C15.4527 13.4833 15.6152 12.1698 15.6152 11H14.1152ZM13.7489 14.4899C13.2677 17.1811 12.4593 19.808 12.2201 20.5576L13.6492 21.0136C13.8946 20.2443 14.7268 17.5432 15.2255 14.7539L13.7489 14.4899ZM20.4229 12.3549C19.6733 12.5941 17.0464 13.4025 14.3552 13.8836L14.6192 15.3602C17.4084 14.8615 20.1096 14.0294 20.8788 13.7839L20.4229 12.3549ZM14.3552 13.8836C13.1274 14.1032 11.9107 14.25 10.8652 14.25V15.75C12.0351 15.75 13.3485 15.5874 14.6192 15.3602L14.3552 13.8836ZM10.8652 14.25C9.81975 14.25 8.6031 14.1032 7.3753 13.8836L7.11129 15.3602C8.38194 15.5874 9.69541 15.75 10.8652 15.75V14.25ZM7.3753 13.8836C4.6841 13.4025 2.0572 12.5941 1.3076 12.3549L0.851632 13.7839C1.62092 14.0294 4.32207 14.8615 7.11129 15.3602L7.3753 13.8836ZM6.11523 11C6.11523 12.1698 6.27782 13.4833 6.50501 14.7539L7.98159 14.4899C7.76206 13.2621 7.61523 12.0455 7.61523 11H6.11523ZM6.50501 14.7539C7.00371 17.5432 7.83586 20.2443 8.08132 21.0136L9.51034 20.5576C9.27116 19.808 8.46277 17.1811 7.98159 14.4899L6.50501 14.7539ZM8.08132 0.986398C7.80127 1.86407 6.74553 5.29348 6.30828 8.48526L7.7944 8.68885C8.21302 5.63305 9.23442 2.30712 9.51034 1.44237L8.08132 0.986398ZM6.30828 8.48526C6.19015 9.34758 6.11523 10.2056 6.11523 11H7.61523C7.61523 10.2943 7.68221 9.5078 7.7944 8.68885L6.30828 8.48526ZM10.8652 8.25C9.76558 8.25 8.47748 8.0876 7.18746 7.84951L6.91522 9.3246C8.24735 9.57046 9.63601 9.75 10.8652 9.75V8.25ZM7.18746 7.84951C5.06013 7.4569 2.9981 6.87202 1.9009 6.53921L1.4655 7.97463C2.58638 8.31462 4.70884 8.9174 6.91522 9.3246L7.18746 7.84951ZM19.7821 6.55831C19.7954 6.55142 19.8113 6.54476 19.8284 6.53956L20.2638 7.97499C20.3377 7.95255 20.4074 7.92382 20.4734 7.88955L19.7821 6.55831ZM1.21255 7.86103C1.29027 7.90856 1.37463 7.94706 1.4655 7.97463L1.9009 6.53921C1.93557 6.54972 1.96753 6.56448 1.99514 6.58136L1.21255 7.86103Z"
                    }
                    fill={"#888888"}
                  ></path>
                </svg>
                <a
                  aria-label={"about"}
                  href={"./aboutus.html"}
                  className={"text-[#868B90]"}
                >
                  {" درباره ما"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img
                  src={"/images/logo/irpsc.irpsc.png"}
                  alt={"زنجیره ایه تامین بهشت"}
                />
                <a href={"https://irpsc.com/"} className={"text-[#868B90]"}>
                  {" زنجیره تامین بهشت"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/3d.irpsc.png"} alt={"سه بعدی متا"} />
                <a href={"https://3d.irpsc.com/"} className={"text-[#868B90]"}>
                  {"سه بعدی متا"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/shop.irpsc.png"} alt={"فروشگاه ملی"} />
                <a
                  href={"https://shop.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"فروشگاه ملی"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img
                  src={"/images/logo/animal.irpsc.png"}
                  alt={"سامانه حیوانات "}
                />
                <a
                  href={"https://animal.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"سامانه حیوانات "}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/ad.irpsc.png"} alt={"تبلیغات ملی"} />
                <a href={"https://ad.irpsc.com"} className={"text-[#868B90]"}>
                  {"تبلیغات ملی"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/art.irpsc.png"} alt={"متا آرت"} />
                <a href={"https://art.irpsc.com/"} className={"text-[#868B90]"}>
                  {"متا آرت"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/faq.irpsc.png"} alt={"انجمن حم"} />
                <a href={"https://faqhub.ir/"} className={"text-[#868B90]"}>
                  {"انجمن حم"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/meta.irpsc.png"} alt={"اخبار متا"} />
                <a
                  href={"https://meta.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"اخبار متا"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/metargb.irpsc.png"} alt={"متارنگ"} />
                <a href={"https://metarang.com/"} className={"text-[#868B90]"}>
                  {"متارنگ"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img
                  src={"/images/logo/supply.irpsc.png"}
                  alt={"تولید کنندگان"}
                />
                <a
                  href={"https://supply.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"تولید کنندگان"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/target.irpsc.png"} alt={"حم "} />
                <a
                  href={"https://target.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"حم"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img
                  src={"/images/logo/uni.irpsc.png"}
                  alt={"دانشگاه متاورس"}
                />
                <a href={"https://uni.irpsc.com/"} className={"text-[#868B90]"}>
                  {"دانشگاه متاورس"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img
                  src={"/images/logo/sale.irpsc.png"}
                  alt={"فروش شماره مجازی "}
                />
                <a
                  href={"https://sale.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"فروش شماره مجازی "}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img
                  src={"/images/logo/video.irpsc.png"}
                  alt={"مرکز آموزش ویدئویی"}
                />
                <a
                  href={"https://video.irpsc.com/"}
                  className={"text-[#868B90]"}
                >
                  {"مرکز آموزش ویدئویی"}
                </a>
              </div>
              <div className={"pr-[20PX] p-[14px] flex items-center gap-4"}>
                <img src={"/images/logo/crm.irpsc.png"} alt={"مدیریت کل"} />
                <a href={"https://crm.irpsc.com/"} className={"text-[#868B90]"}>
                  {" مدیریت کل "}
                </a>
              </div>
              <div>
                <ul className={"tree"}>
                  <li className={"flex flex-col gap-3"}>
                    <input type={"checkbox"} id={"c11"} className={"peer"} />
                    <label
                      className={
                        "px-[20px] py-4 w-full rounded-[10px] text-[#868B90]  peer-checked:text-white head_label peer-checked:bg-[#000BEE] dark:peer-checked:bg-[#C2008C] peer-checked:[&>div>svg]:rotate-180 peer-checked:[&>div>svg>path]:stroke-white"
                      }
                      htmlFor={"c11"}
                    >
                      <div
                        className={
                          "flex w-full justify-between items-center select-none"
                        }
                      >
                        <div className={"flex items-center gap-4"}>
                          <svg
                            width={"22"}
                            height={"22"}
                            viewBox={"0 0 22 22"}
                            fill={"none"}
                            xmlns={"http://www.w3.org/2000/svg"}
                          >
                            <path
                              d={
                                "M6.86523 11H6.11523H6.86523ZM14.8652 11H15.6152H14.8652ZM10.8652 15V15.75V15ZM12.9346 20.7856L13.0891 21.5195L12.9346 20.7856ZM8.79583 20.7856L8.6414 21.5195L8.79583 20.7856ZM1.60384 7.22119L0.909524 6.93761H0.909524L1.60384 7.22119ZM1.07962 13.0694L0.345688 13.2238L1.07962 13.0694ZM8.79583 1.21438L8.6414 0.480454L8.79583 1.21438ZM12.9346 1.21438L13.0891 0.480455L12.9346 1.21438ZM20.0461 7.25728L20.2638 7.97499L20.0461 7.25728ZM1.6832 7.25692L1.9009 6.53921L1.6832 7.25692ZM19.4333 7.50731C19.8727 8.58399 20.1152 9.76274 20.1152 11H21.6152C21.6152 9.56509 21.3337 8.19407 20.8221 6.94055L19.4333 7.50731ZM20.1152 11C20.1152 11.6573 20.0468 12.2978 19.9169 12.915L21.3848 13.2238C21.5359 12.5057 21.6152 11.7617 21.6152 11H20.1152ZM19.9169 12.915C19.1655 16.4862 16.3514 19.3002 12.7802 20.0517L13.0891 21.5195C17.2422 20.6456 20.5109 17.377 21.3848 13.2238L19.9169 12.915ZM12.7802 20.0517C12.163 20.1816 11.5225 20.25 10.8652 20.25V21.75C11.6269 21.75 12.3709 21.6707 13.0891 21.5195L12.7802 20.0517ZM10.8652 20.25C10.208 20.25 9.56748 20.1816 8.95026 20.0517L8.6414 21.5195C9.35957 21.6707 10.1035 21.75 10.8652 21.75V20.25ZM1.61523 11C1.61523 9.76178 1.85813 8.58215 2.29816 7.50478L0.909524 6.93761C0.397214 8.19193 0.115235 9.56397 0.115235 11H1.61523ZM8.95026 20.0517C5.37906 19.3002 2.565 16.4862 1.81354 12.915L0.345688 13.2238C1.21959 17.377 4.48828 20.6456 8.6414 21.5195L8.95026 20.0517ZM1.81354 12.915C1.68367 12.2978 1.61523 11.6573 1.61523 11H0.115235C0.115235 11.7617 0.19457 12.5057 0.345688 13.2238L1.81354 12.915ZM2.29816 7.50478C3.44348 4.70064 5.92609 2.58466 8.95026 1.94831L8.6414 0.480454C5.1226 1.22088 2.2401 3.6799 0.909524 6.93761L2.29816 7.50478ZM8.95026 1.94831C9.56748 1.81844 10.208 1.75 10.8652 1.75V0.25C10.1035 0.25 9.35957 0.329336 8.6414 0.480454L8.95026 1.94831ZM10.8652 1.75C11.5225 1.75 12.163 1.81844 12.7802 1.94831L13.0891 0.480455C12.3709 0.329336 11.6269 0.25 10.8652 0.25V1.75ZM12.7802 1.94831C15.8053 2.58485 18.2885 4.70192 19.4333 7.50731L20.8221 6.94055C19.4921 3.68138 16.6089 1.22111 13.0891 0.480455L12.7802 1.94831ZM12.2201 1.44237C12.4961 2.30712 13.5174 5.63305 13.9361 8.68885L15.4222 8.48526C14.9849 5.29348 13.9292 1.86407 13.6492 0.986399L12.2201 1.44237ZM13.9361 8.68885C14.0483 9.5078 14.1152 10.2943 14.1152 11H15.6152C15.6152 10.2056 15.5403 9.34758 15.4222 8.48526L13.9361 8.68885ZM19.8284 6.53956C18.7309 6.87243 16.6696 7.45704 14.543 7.84951L14.8153 9.3246C17.0208 8.91754 19.1426 8.31505 20.2638 7.97499L19.8284 6.53956ZM14.543 7.84951C13.253 8.0876 11.9649 8.25 10.8652 8.25V9.75C12.0945 9.75 13.4831 9.57046 14.8153 9.3246L14.543 7.84951ZM14.1152 11C14.1152 12.0455 13.9684 13.2621 13.7489 14.4899L15.2255 14.7539C15.4527 13.4833 15.6152 12.1698 15.6152 11H14.1152ZM13.7489 14.4899C13.2677 17.1811 12.4593 19.808 12.2201 20.5576L13.6492 21.0136C13.8946 20.2443 14.7268 17.5432 15.2255 14.7539L13.7489 14.4899ZM20.4229 12.3549C19.6733 12.5941 17.0464 13.4025 14.3552 13.8836L14.6192 15.3602C17.4084 14.8615 20.1096 14.0294 20.8788 13.7839L20.4229 12.3549ZM14.3552 13.8836C13.1274 14.1032 11.9107 14.25 10.8652 14.25V15.75C12.0351 15.75 13.3485 15.5874 14.6192 15.3602L14.3552 13.8836ZM10.8652 14.25C9.81975 14.25 8.6031 14.1032 7.3753 13.8836L7.11129 15.3602C8.38194 15.5874 9.69541 15.75 10.8652 15.75V14.25ZM7.3753 13.8836C4.6841 13.4025 2.0572 12.5941 1.3076 12.3549L0.851632 13.7839C1.62092 14.0294 4.32207 14.8615 7.11129 15.3602L7.3753 13.8836ZM6.11523 11C6.11523 12.1698 6.27782 13.4833 6.50501 14.7539L7.98159 14.4899C7.76206 13.2621 7.61523 12.0455 7.61523 11H6.11523ZM6.50501 14.7539C7.00371 17.5432 7.83586 20.2443 8.08132 21.0136L9.51034 20.5576C9.27116 19.808 8.46277 17.1811 7.98159 14.4899L6.50501 14.7539ZM8.08132 0.986398C7.80127 1.86407 6.74553 5.29348 6.30828 8.48526L7.7944 8.68885C8.21302 5.63305 9.23442 2.30712 9.51034 1.44237L8.08132 0.986398ZM6.30828 8.48526C6.19015 9.34758 6.11523 10.2056 6.11523 11H7.61523C7.61523 10.2943 7.68221 9.5078 7.7944 8.68885L6.30828 8.48526ZM10.8652 8.25C9.76558 8.25 8.47748 8.0876 7.18746 7.84951L6.91522 9.3246C8.24735 9.57046 9.63601 9.75 10.8652 9.75V8.25ZM7.18746 7.84951C5.06013 7.4569 2.9981 6.87202 1.9009 6.53921L1.4655 7.97463C2.58638 8.31462 4.70884 8.9174 6.91522 9.3246L7.18746 7.84951ZM19.7821 6.55831C19.7954 6.55142 19.8113 6.54476 19.8284 6.53956L20.2638 7.97499C20.3377 7.95255 20.4074 7.92382 20.4734 7.88955L19.7821 6.55831ZM1.21255 7.86103C1.29027 7.90856 1.37463 7.94706 1.4655 7.97463L1.9009 6.53921C1.93557 6.54972 1.96753 6.56448 1.99514 6.58136L1.21255 7.86103Z"
                              }
                              fill={"#888888"}
                            ></path>
                          </svg>
                          <p>{" زبان "}</p>
                        </div>
                        <svg
                          className={"transition-[5s] duration-300"}
                          width={"15"}
                          height={"9"}
                          viewBox={"0 0 15 9"}
                          fill={"none"}
                          xmlns={"http://www.w3.org/2000/svg"}
                        >
                          <path
                            className={" stroke-[#868B90]"}
                            d={"M14 1L7.5 7.5L1 0.999999"}
                            stroke={"black"}
                            strokeWidth={"2"}
                            strokeLinecap={"round"}
                            strokeLinejoin={"round"}
                          ></path>
                        </svg>
                      </div>
                    </label>
                    <ul>
                      <div
                        id={"zaban"}
                        className={
                          "flex flex-col text-sm text-[#000BEE] font-bold gap-5 p-3 pr-[20PX] dark:text-[#868B90]"
                        }
                      >
                        <a href={"#"}>{"فارسی"}</a>
                        <a href={"#"}>{" انگلیسی"}</a>
                      </div>
                    </ul>
                  </li>
                </ul>
              </div>
            </div>
            <div
              className={
                "space-y-6  lg:absolute bottom-0 lg:h-[150px]  w-full h-auto  bg-white  dark:bg-[#0F0F0E] pb-10 lg:pb-1 pt-5"
              }
            >
              <SessionControl />
              <div
                className={
                  "pt-5 border-t-2 border-[#EFEFEF]   dark:border-[#868B90] "
                }
              >
                <div
                  className={
                    " flex rounded-full w-full p-[6px] bg-[#F4F4F4] dark:bg-[#090909] "
                  }
                >
                  <button
                    aria-label={"theme"}
                    onClick={() => setTheme(true)}
                    className={
                      "enable-dark-mode bg-transparent dark:bg-[#0F0F0E] flex justify-center p-1 rounded-full w-1/2  "
                    }
                  >
                    <div>
                      <svg
                        width={"16"}
                        height={"16"}
                        xmlns={"http://www.w3.org/2000/svg"}
                      >
                        <path
                          className={"fill-slate-400 dark:fill-white"}
                          d={
                            "M6.2 1C3.2 1.8 1 4.6 1 7.9 1 11.8 4.2 15 8.1 15c3.3 0 6-2.2 6.9-5.2C9.7 11.2 4.8 6.3 6.2 1Z"
                          }
                        ></path>
                        <path
                          className={"fill-slate-500 dark:fill-white"}
                          d={
                            "M12.5 5a.625.625 0 0 1-.625-.625 1.252 1.252 0 0 0-1.25-1.25.625.625 0 1 1 0-1.25 1.252 1.252 0 0 0 1.25-1.25.625.625 0 1 1 1.25 0c.001.69.56 1.249 1.25 1.25a.625.625 0 1 1 0 1.25c-.69.001-1.249.56-1.25 1.25A.625.625 0 0 1 12.5 5Z"
                          }
                        ></path>
                      </svg>
                    </div>
                  </button>
                  <button
                    aria-label={"theme"}
                    onClick={() => setTheme(false)}
                    className={
                      "disable-dark-mode  bg-[#FCFCFC] dark:bg-transparent flex justify-center p-1 rounded-full w-1/2 shadow-[0_0_6px_0_rgba(0,0,0,0.1)] dark:shadow-none"
                    }
                  >
                    <div>
                      <svg
                        width={"16"}
                        height={"16"}
                        xmlns={"http://www.w3.org/2000/svg"}
                      >
                        <path
                          className={"fill-slate-300 dark:fill-white"}
                          d={
                            "M7 0h2v2H7zM12.88 1.637l1.414 1.415-1.415 1.413-1.413-1.414zM14 7h2v2h-2zM12.95 14.433l-1.414-1.413 1.413-1.415 1.415 1.414zM7 14h2v2H7zM2.98 14.364l-1.413-1.415 1.414-1.414 1.414 1.415zM0 7h2v2H0zM3.05 1.706 4.463 3.12 3.05 4.535 1.636 3.12z"
                          }
                        ></path>
                        <path
                          className={"fill-slate-400 dark:fill-white"}
                          d={
                            "M8 4C5.8 4 4 5.8 4 8s1.8 4 4 4 4-1.8 4-4-1.8-4-4-4Z"
                          }
                        ></path>
                      </svg>
                    </div>
                  </button>
                </div>
              </div>
            </div>
          </nav>
        </div>

        <div dir={"ltr"} id={"close00"} className="h-full w-full min-w-0">
          <nav
            dir={"rtl"}
            className={
              "bg-[#FCFCFC] dark:bg-[#0F0F0E] flex items-center flex-col h-full w-full min-w-0 relative"
            }
          >
            <div
              className={
                "flex items-center flex-col gap-11 flex-1 min-h-0 w-full pb-6 relative overflow-y-auto overflow-x-hidden scrollbar"
              }
            >
              <div
                className={
                  "w-full shrink-0 gap-6 flex flex-col items-center justify-center px-3 pt-6 sticky top-0 z-10 bg-[#FCFCFC] dark:bg-[#0F0F0E]"
                }
              >
                <div
                  id={"open-nav-btn"}
                  className={"items-center w-6 h-6"}
                  onClick={openNav}
                  role="button"
                  tabIndex={0}
                  aria-label="باز کردن فهرست"
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      openNav();
                    }
                  }}
                >
                  <svg
                    className={" dark:fill-white"}
                    width={"30"}
                    height={"22"}
                    viewBox={"0 0 30 22"}
                    fill={"none"}
                    xmlns={"http://www.w3.org/2000/svg"}
                  >
                    <path
                      className={"dark:fill-white"}
                      fillRule={"evenodd"}
                      clipRule={"evenodd"}
                      d={
                        "M0 1.375C0 0.615608 0.6296 0 1.40625 0H28.5938C29.3704 0 30 0.615608 30 1.375C30 2.13439 29.3704 2.75 28.5938 2.75H1.40625C0.6296 2.75 0 2.13439 0 1.375ZM0 11C0 10.2406 0.6296 9.625 1.40625 9.625H28.5938C29.3704 9.625 30 10.2406 30 11C30 11.7594 29.3704 12.375 28.5938 12.375H1.40625C0.6296 12.375 0 11.7594 0 11ZM0 20.625C0 19.8656 0.6296 19.25 1.40625 19.25H28.5938C29.3704 19.25 30 19.8656 30 20.625C30 21.3844 29.3704 22 28.5938 22H1.40625C0.6296 22 0 21.3844 0 20.625Z"
                      }
                      fill={"#0713EF"}
                    ></path>
                  </svg>
                </div>

                <div className={"w-6  flex items-center justify-center"}>
                  <a aria-label={"logo"} href={"/home"} className={"w-full"}>
                    <img
                      src={"/images/logo/accounts.irpsc.png"}
                      alt={""}
                      className={"w-full"}
                    />
                  </a>
                </div>
                <hr className="sidebar-divider" />
              </div>
              <div className={" w-6 h-6  "}>
                <a href={"https://irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/irpsc.irpsc.png"}
                    alt={"irpsc"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={" w-6 h-6  "}>
                <a href={"https://3d.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/3d.irpsc.png"}
                    alt={"3dmeta"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={" w-6 h-6  "}>
                <a href={"https://shop.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/shop.irpsc.png"}
                    alt={"shop"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={" w-6 h-6  "}>
                <a href={"https://animal.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/animal.irpsc.png"}
                    alt={"animal"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={"w-6 h-6"}>
                <a href={"https://ad.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/ad.irpsc.png"}
                    alt={"ad irpsc"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={"w-6 h-6 "}>
                <a href={"https://art.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/art.irpsc.png"}
                    alt={"meta art"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={"w-6 h-6"}>
                <a href={"https://faq.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/faq.irpsc.png"}
                    alt={"faq"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={"w-6 h-6"}>
                <a href={"https://meta.irpsc.com/"} className={"w-full"}>
                  <img
                    src={"/images/logo/meta.irpsc.png"}
                    alt={"meta"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={"w-6 h-6"}>
                <a href={"https://metarang.com"} className={"w-full"}>
                  <img
                    src={"/images/logo/metargb.irpsc.png"}
                    alt={"metargb"}
                    className={"w-6 h-6"}
                  />
                </a>
              </div>
              <div className={"w-6 h-6"}>
                <a aria-label={"lang"} href={"#"} className={"w-full"}>
                  <svg
                    width={"22"}
                    height={"22"}
                    viewBox={"0 0 22 22"}
                    fill={"none"}
                    xmlns={"http://www.w3.org/2000/svg"}
                  >
                    <path
                      d={
                        "M6.86523 11H6.11523H6.86523ZM14.8652 11H15.6152H14.8652ZM10.8652 15V15.75V15ZM12.9346 20.7856L13.0891 21.5195L12.9346 20.7856ZM8.79583 20.7856L8.6414 21.5195L8.79583 20.7856ZM1.60384 7.22119L0.909524 6.93761H0.909524L1.60384 7.22119ZM1.07962 13.0694L0.345688 13.2238L1.07962 13.0694ZM8.79583 1.21438L8.6414 0.480454L8.79583 1.21438ZM12.9346 1.21438L13.0891 0.480455L12.9346 1.21438ZM20.0461 7.25728L20.2638 7.97499L20.0461 7.25728ZM1.6832 7.25692L1.9009 6.53921L1.6832 7.25692ZM19.4333 7.50731C19.8727 8.58399 20.1152 9.76274 20.1152 11H21.6152C21.6152 9.56509 21.3337 8.19407 20.8221 6.94055L19.4333 7.50731ZM20.1152 11C20.1152 11.6573 20.0468 12.2978 19.9169 12.915L21.3848 13.2238C21.5359 12.5057 21.6152 11.7617 21.6152 11H20.1152ZM19.9169 12.915C19.1655 16.4862 16.3514 19.3002 12.7802 20.0517L13.0891 21.5195C17.2422 20.6456 20.5109 17.377 21.3848 13.2238L19.9169 12.915ZM12.7802 20.0517C12.163 20.1816 11.5225 20.25 10.8652 20.25V21.75C11.6269 21.75 12.3709 21.6707 13.0891 21.5195L12.7802 20.0517ZM10.8652 20.25C10.208 20.25 9.56748 20.1816 8.95026 20.0517L8.6414 21.5195C9.35957 21.6707 10.1035 21.75 10.8652 21.75V20.25ZM1.61523 11C1.61523 9.76178 1.85813 8.58215 2.29816 7.50478L0.909524 6.93761C0.397214 8.19193 0.115235 9.56397 0.115235 11H1.61523ZM8.95026 20.0517C5.37906 19.3002 2.565 16.4862 1.81354 12.915L0.345688 13.2238C1.21959 17.377 4.48828 20.6456 8.6414 21.5195L8.95026 20.0517ZM1.81354 12.915C1.68367 12.2978 1.61523 11.6573 1.61523 11H0.115235C0.115235 11.7617 0.19457 12.5057 0.345688 13.2238L1.81354 12.915ZM2.29816 7.50478C3.44348 4.70064 5.92609 2.58466 8.95026 1.94831L8.6414 0.480454C5.1226 1.22088 2.2401 3.6799 0.909524 6.93761L2.29816 7.50478ZM8.95026 1.94831C9.56748 1.81844 10.208 1.75 10.8652 1.75V0.25C10.1035 0.25 9.35957 0.329336 8.6414 0.480454L8.95026 1.94831ZM10.8652 1.75C11.5225 1.75 12.163 1.81844 12.7802 1.94831L13.0891 0.480455C12.3709 0.329336 11.6269 0.25 10.8652 0.25V1.75ZM12.7802 1.94831C15.8053 2.58485 18.2885 4.70192 19.4333 7.50731L20.8221 6.94055C19.4921 3.68138 16.6089 1.22111 13.0891 0.480455L12.7802 1.94831ZM12.2201 1.44237C12.4961 2.30712 13.5174 5.63305 13.9361 8.68885L15.4222 8.48526C14.9849 5.29348 13.9292 1.86407 13.6492 0.986399L12.2201 1.44237ZM13.9361 8.68885C14.0483 9.5078 14.1152 10.2943 14.1152 11H15.6152C15.6152 10.2056 15.5403 9.34758 15.4222 8.48526L13.9361 8.68885ZM19.8284 6.53956C18.7309 6.87243 16.6696 7.45704 14.543 7.84951L14.8153 9.3246C17.0208 8.91754 19.1426 8.31505 20.2638 7.97499L19.8284 6.53956ZM14.543 7.84951C13.253 8.0876 11.9649 8.25 10.8652 8.25V9.75C12.0945 9.75 13.4831 9.57046 14.8153 9.3246L14.543 7.84951ZM14.1152 11C14.1152 12.0455 13.9684 13.2621 13.7489 14.4899L15.2255 14.7539C15.4527 13.4833 15.6152 12.1698 15.6152 11H14.1152ZM13.7489 14.4899C13.2677 17.1811 12.4593 19.808 12.2201 20.5576L13.6492 21.0136C13.8946 20.2443 14.7268 17.5432 15.2255 14.7539L13.7489 14.4899ZM20.4229 12.3549C19.6733 12.5941 17.0464 13.4025 14.3552 13.8836L14.6192 15.3602C17.4084 14.8615 20.1096 14.0294 20.8788 13.7839L20.4229 12.3549ZM14.3552 13.8836C13.1274 14.1032 11.9107 14.25 10.8652 14.25V15.75C12.0351 15.75 13.3485 15.5874 14.6192 15.3602L14.3552 13.8836ZM10.8652 14.25C9.81975 14.25 8.6031 14.1032 7.3753 13.8836L7.11129 15.3602C8.38194 15.5874 9.69541 15.75 10.8652 15.75V14.25ZM7.3753 13.8836C4.6841 13.4025 2.0572 12.5941 1.3076 12.3549L0.851632 13.7839C1.62092 14.0294 4.32207 14.8615 7.11129 15.3602L7.3753 13.8836ZM6.11523 11C6.11523 12.1698 6.27782 13.4833 6.50501 14.7539L7.98159 14.4899C7.76206 13.2621 7.61523 12.0455 7.61523 11H6.11523ZM6.50501 14.7539C7.00371 17.5432 7.83586 20.2443 8.08132 21.0136L9.51034 20.5576C9.27116 19.808 8.46277 17.1811 7.98159 14.4899L6.50501 14.7539ZM8.08132 0.986398C7.80127 1.86407 6.74553 5.29348 6.30828 8.48526L7.7944 8.68885C8.21302 5.63305 9.23442 2.30712 9.51034 1.44237L8.08132 0.986398ZM6.30828 8.48526C6.19015 9.34758 6.11523 10.2056 6.11523 11H7.61523C7.61523 10.2943 7.68221 9.5078 7.7944 8.68885L6.30828 8.48526ZM10.8652 8.25C9.76558 8.25 8.47748 8.0876 7.18746 7.84951L6.91522 9.3246C8.24735 9.57046 9.63601 9.75 10.8652 9.75V8.25ZM7.18746 7.84951C5.06013 7.4569 2.9981 6.87202 1.9009 6.53921L1.4655 7.97463C2.58638 8.31462 4.70884 8.9174 6.91522 9.3246L7.18746 7.84951ZM19.7821 6.55831C19.7954 6.55142 19.8113 6.54476 19.8284 6.53956L20.2638 7.97499C20.3377 7.95255 20.4074 7.92382 20.4734 7.88955L19.7821 6.55831ZM1.21255 7.86103C1.29027 7.90856 1.37463 7.94706 1.4655 7.97463L1.9009 6.53921C1.93557 6.54972 1.96753 6.56448 1.99514 6.58136L1.21255 7.86103Z"
                      }
                      fill={"#888888"}
                    ></path>
                  </svg>
                </a>
              </div>
            </div>
            <div
              className={
                "w-full shrink-0 space-y-6 px-3 bg-[#FCFCFC] dark:bg-[#0F0F0E] py-5"
              }
            >
              <SessionControl />
              <hr className="sidebar-divider" />
              <div className={"flex justify-center"}>
                <div
                  className={
                    " flex rounded-full w-max p-[6px] bg-[#F4F4F4] dark:bg-[#090909]"
                  }
                >
                  <button
                    aria-label={"theme"}
                    onClick={() => setTheme(true)}
                    className={
                      "enable-dark-mode2 dark:hidden bg-transparent dark:bg-[#0F0F0E] flex justify-center items-center p-1 rounded-full w-5 h-5  "
                    }
                  >
                    <div>
                      <svg
                        width={"16"}
                        height={"16"}
                        xmlns={"http://www.w3.org/2000/svg"}
                      >
                        <path
                          className={"fill-slate-400 dark:fill-white"}
                          d={
                            "M6.2 1C3.2 1.8 1 4.6 1 7.9 1 11.8 4.2 15 8.1 15c3.3 0 6-2.2 6.9-5.2C9.7 11.2 4.8 6.3 6.2 1Z"
                          }
                        ></path>
                        <path
                          className={"fill-slate-500 dark:fill-white"}
                          d={
                            "M12.5 5a.625.625 0 0 1-.625-.625 1.252 1.252 0 0 0-1.25-1.25.625.625 0 1 1 0-1.25 1.252 1.252 0 0 0 1.25-1.25.625.625 0 1 1 1.25 0c.001.69.56 1.249 1.25 1.25a.625.625 0 1 1 0 1.25c-.69.001-1.249.56-1.25 1.25A.625.625 0 0 1 12.5 5Z"
                          }
                        ></path>
                      </svg>
                    </div>
                  </button>
                  <button
                    aria-label={"theme"}
                    onClick={() => setTheme(false)}
                    className={
                      "disable-dark-mode2 hidden   bg-[#FCFCFC] dark:bg-transparent dark:flex justify-center items-center p-1 rounded-full w-5 h-5  shadow-[0_0_6px_0_rgba(0,0,0,0.1)] dark:shadow-none"
                    }
                  >
                    <div>
                      <svg
                        width={"16"}
                        height={"16"}
                        xmlns={"http://www.w3.org/2000/svg"}
                      >
                        <path
                          className={"fill-slate-300 dark:fill-white"}
                          d={
                            "M7 0h2v2H7zM12.88 1.637l1.414 1.415-1.415 1.413-1.413-1.414zM14 7h2v2h-2zM12.95 14.433l-1.414-1.413 1.413-1.415 1.415 1.414zM7 14h2v2H7zM2.98 14.364l-1.413-1.415 1.414-1.414 1.414 1.415zM0 7h2v2H0zM3.05 1.706 4.463 3.12 3.05 4.535 1.636 3.12z"
                          }
                        ></path>
                        <path
                          className={"fill-slate-400 dark:fill-white"}
                          d={
                            "M8 4C5.8 4 4 5.8 4 8s1.8 4 4 4 4-1.8 4-4-1.8-4-4-4Z"
                          }
                        ></path>
                      </svg>
                    </div>
                  </button>
                </div>
              </div>
            </div>
          </nav>
        </div>
      </div>
    </header>
  );
}
